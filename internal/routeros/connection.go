package routeros

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3"
	"gopkg.in/yaml.v3"
)

// Mikrotik silently reaps idle API sessions; without TCP keepalive the next
// scrape after an idle gap fails with "connection closed".
const tcpKeepAlive = 30 * time.Second

type BackoffConfig struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Divisor      int
}

var DefaultBackoff = BackoffConfig{
	InitialDelay: 10 * time.Second,
	MaxDelay:     900 * time.Second,
	Divisor:      5,
}

type ConnectionConfig struct {
	RouterName           string
	Hostname             string
	Port                 int
	Username             string
	Password             string
	CredentialsFile      string
	PlaintextLogin       bool
	UseSSL               bool
	NoSSLCertificate     bool
	SSLCertificateVerify bool
	SSLCheckHostname     bool
	SSLCAFile            string
	SocketTimeout        time.Duration
	Backoff              BackoffConfig
}

type credentialsFile struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// apiClient is the subset of *routeros.Client used by Connection.
type apiClient interface {
	RunArgsContext(ctx context.Context, sentence []string) (*routeros.Reply, error)
	ListenArgsQueue(sentence []string, queueSize int) (*routeros.ListenReply, error)
	Close() error
}

// Connection holds two API sessions per router so RunStream's async mode
// can't interfere with Run. See acquireStreamClient for why.
type Connection struct {
	cfg ConnectionConfig

	mu           sync.Mutex
	runClient    apiClient
	streamClient apiClient
	lastFailure  time.Time
	failureCount int
}

func NewConnection(cfg ConnectionConfig) *Connection {
	return &Connection{cfg: cfg}
}

func (c *Connection) RouterName() string {
	return c.cfg.RouterName
}

func (c *Connection) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.runClient != nil
}

func (c *Connection) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.runClient != nil {
		return nil
	}

	if c.inBackoff(ctx, time.Now()) {
		return nil
	}

	slog.Info("Connecting to router", "name", c.cfg.RouterName, "host", c.cfg.Hostname)

	username, password, err := c.resolveCredentials()
	if err != nil {
		c.recordFailure(time.Now())
		return err
	}

	client, err := c.dial(ctx, username, password)
	if err != nil {
		c.recordFailure(time.Now())
		return fmt.Errorf("routers: connect %s@%s: %w", c.cfg.RouterName, c.cfg.Hostname, err)
	}

	c.runClient = client
	c.failureCount = 0
	c.lastFailure = time.Time{}
	slog.Info("Connection established", "name", c.cfg.RouterName, "host", c.cfg.Hostname)
	return nil
}

func (c *Connection) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeAllLocked()
}

// closeAllLocked closes both clients. Caller must hold c.mu.
func (c *Connection) closeAllLocked() {
	if c.runClient != nil {
		_ = c.runClient.Close()
		c.runClient = nil
	}
	if c.streamClient != nil {
		_ = c.streamClient.Close()
		c.streamClient = nil
	}
}

// failRunLocked closes only runClient if it's still the same instance we
// observed. Stream-side state is preserved so a healthy in-flight RunStream
// isn't mid-cancelled by an unrelated Run failure.
func (c *Connection) failRunLocked(cl apiClient) {
	if c.runClient == cl {
		_ = c.runClient.Close()
		c.runClient = nil
		c.recordFailure(time.Now())
	}
}

// failStream closes only streamClient if it's still cl. Symmetric to
// failRunLocked.
func (c *Connection) failStream(cl apiClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.streamClient == cl {
		_ = c.streamClient.Close()
		c.streamClient = nil
		c.recordFailure(time.Now())
	}
}

func (c *Connection) Run(ctx context.Context, sentence ...string) ([]map[string]string, error) {
	c.mu.Lock()
	cl := c.runClient

	if cl == nil {
		c.mu.Unlock()
		return nil, fmt.Errorf("routers: not connected to %s@%s", c.cfg.RouterName, c.cfg.Hostname)
	}

	reply, err := cl.RunArgsContext(ctx, sentence)
	if err != nil {
		// ctx cancel means the caller gave up; don't tear down a healthy socket.
		if !isContextError(err) {
			c.failRunLocked(cl)
		}
		c.mu.Unlock()
		return nil, err
	}
	c.mu.Unlock()

	result := make([]map[string]string, 0, len(reply.Re))
	for _, s := range reply.Re {
		result = append(result, s.Map)
	}

	if len(result) == 0 && reply.Done != nil && len(reply.Done.Map) > 0 {
		result = append(result, reply.Done.Map)
	}

	return result, nil
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func (c *Connection) dial(ctx context.Context, username, password string) (*routeros.Client, error) {
	addr := fmt.Sprintf("%s:%d", c.cfg.Hostname, c.cfg.Port)
	timeout := c.cfg.SocketTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	netDialer := &net.Dialer{KeepAlive: tcpKeepAlive}

	var conn net.Conn
	var err error
	if c.cfg.UseSSL {
		tlsCfg := &tls.Config{ServerName: c.cfg.Hostname}
		if c.cfg.NoSSLCertificate || !c.cfg.SSLCertificateVerify || !c.cfg.SSLCheckHostname {
			var insecureReasons []string
			if c.cfg.NoSSLCertificate {
				insecureReasons = append(insecureReasons, "NoSSLCertificate")
			}
			if !c.cfg.SSLCertificateVerify {
				insecureReasons = append(insecureReasons, "SSLCertificateVerify disabled")
			}
			if !c.cfg.SSLCheckHostname {
				insecureReasons = append(insecureReasons, "SSLCheckHostname disabled")
			}
			slog.Error("TLS security disabled - certificate verification bypassed",
				"host", c.cfg.Hostname,
				"reasons", insecureReasons)
			tlsCfg.InsecureSkipVerify = true //nolint:gosec
		} else {
			slog.Info("Connecting with TLS (secure mode)", "host", c.cfg.Hostname)
		}
		conn, err = (&tls.Dialer{NetDialer: netDialer, Config: tlsCfg}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = netDialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return nil, err
	}

	client, err := routeros.NewClient(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := client.LoginContext(ctx, username, password); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

func (c *Connection) RunStream(ctx context.Context, callback func(map[string]string), sentence ...string) error {
	cl, err := c.acquireStreamClient(ctx)
	if err != nil {
		return err
	}

	l, err := cl.ListenArgsQueue(sentence, 1000)
	if err != nil {
		if !isContextError(err) {
			c.failStream(cl)
		}
		return err
	}

	ctxChan := ctx.Done()
	var cancelOnce sync.Once

	for {
		select {
		case <-ctxChan:
			// Context timed out. Disable this select case and tell the router to abort.
			ctxChan = nil
			cancelOnce.Do(func() {
				go func() {
					_, _ = l.Cancel()
				}()
			})

		case sen, ok := <-l.Chan():
			if !ok {
				// Stream finished or connection dropped.
				if err := l.Err(); err != nil {
					if !isContextError(err) {
						c.failStream(cl)
					}
					return err
				}
				return ctx.Err()
			}

			// Process the record if we haven't timed out.
			if ctxChan != nil && sen != nil && len(sen.Map) > 0 {
				callback(sen.Map)
			}
		}
	}
}

// acquireStreamClient returns the streamClient, dialing a second API session
// on first use. Dialing happens *without* c.mu held — a TLS handshake +
// LoginContext can take seconds and would otherwise block every Run on the
// same Connection. On race we close the loser and use the winner.
func (c *Connection) acquireStreamClient(ctx context.Context) (apiClient, error) {
	c.mu.Lock()
	if c.streamClient != nil {
		cl := c.streamClient
		c.mu.Unlock()
		return cl, nil
	}
	if c.runClient == nil {
		c.mu.Unlock()
		return nil, fmt.Errorf("routers: not connected to %s@%s", c.cfg.RouterName, c.cfg.Hostname)
	}
	username, password, err := c.resolveCredentials()
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}

	client, err := c.dial(ctx, username, password)
	if err != nil {
		c.mu.Lock()
		c.recordFailure(time.Now())
		c.mu.Unlock()
		return nil, fmt.Errorf("routers: stream connect %s@%s: %w", c.cfg.RouterName, c.cfg.Hostname, err)
	}

	c.mu.Lock()
	if c.streamClient != nil {
		// Lost the race; another goroutine already dialed.
		winner := c.streamClient
		c.mu.Unlock()
		_ = client.Close()
		return winner, nil
	}
	c.streamClient = client
	c.mu.Unlock()
	return client, nil
}

func (c *Connection) resolveCredentials() (string, string, error) {
	if c.cfg.CredentialsFile == "" {
		return c.cfg.Username, c.cfg.Password, nil
	}
	data, err := os.ReadFile(c.cfg.CredentialsFile)
	if err != nil {
		return "", "", fmt.Errorf("routers: read credentials file %q: %w", c.cfg.CredentialsFile, err)
	}
	var creds credentialsFile
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return "", "", fmt.Errorf("routers: parse credentials file %q: %w", c.cfg.CredentialsFile, err)
	}
	username := c.cfg.Username
	password := c.cfg.Password
	if creds.Username != "" {
		username = creds.Username
	}
	if creds.Password != "" {
		password = creds.Password
	}
	return username, password, nil
}

func (c *Connection) inBackoff(ctx context.Context, now time.Time) bool {
	if c.lastFailure.IsZero() {
		return false
	}
	delay := c.connectDelay()
	remaining := delay - now.Sub(c.lastFailure)
	if remaining > 0 {
		slog.InfoContext(ctx, "Router in connect backoff",
			"name", c.cfg.RouterName,
			"remaining_secs", int(remaining.Seconds()),
			"failures", c.failureCount)
		return true
	}
	return false
}

func (c *Connection) connectDelay() time.Duration {
	b := c.cfg.Backoff
	if b.Divisor <= 0 {
		b = DefaultBackoff
	}
	factor := 1.0 + float64(c.failureCount)/float64(b.Divisor)
	delay := time.Duration(float64(b.InitialDelay) * factor)
	if delay > b.MaxDelay {
		delay = b.MaxDelay
	}
	return delay
}

func (c *Connection) recordFailure(now time.Time) {
	c.failureCount++
	c.lastFailure = now
}
