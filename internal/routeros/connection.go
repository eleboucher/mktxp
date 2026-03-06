package routeros

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3"
	"gopkg.in/yaml.v3"
)

type BackoffConfig struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Divisor      int
}

var DefaultBackoff = BackoffConfig{
	InitialDelay: 120 * time.Second,
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

type Connection struct {
	cfg ConnectionConfig

	mu           sync.Mutex
	client       *routeros.Client
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
	return c.client != nil
}

func (c *Connection) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
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

	c.client = client
	c.failureCount = 0
	c.lastFailure = time.Time{}
	slog.Info("Connection established", "name", c.cfg.RouterName, "host", c.cfg.Hostname)
	return nil
}

func (c *Connection) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		_ = c.client.Close()
		c.client = nil
	}
}

func (c *Connection) Run(ctx context.Context, sentence ...string) ([]map[string]string, error) {
	c.mu.Lock()
	cl := c.client
	defer c.mu.Unlock()

	if cl == nil {
		return nil, fmt.Errorf("routers: not connected to %s@%s", c.cfg.RouterName, c.cfg.Hostname)
	}

	reply, err := cl.RunArgsContext(ctx, sentence)
	if err != nil {
		c.mu.Lock()
		if c.client == cl {
			_ = c.client.Close()
			c.client = nil
			c.recordFailure(time.Now())
		}
		c.mu.Unlock()
		return nil, err
	}

	result := make([]map[string]string, 0, len(reply.Re))
	for _, s := range reply.Re {
		result = append(result, s.Map)
	}

	if len(result) == 0 && reply.Done != nil && len(reply.Done.Map) > 0 {
		result = append(result, reply.Done.Map)
	}

	return result, nil
}

func (c *Connection) dial(ctx context.Context, username, password string) (*routeros.Client, error) {
	addr := fmt.Sprintf("%s:%d", c.cfg.Hostname, c.cfg.Port)
	timeout := c.cfg.SocketTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if !c.cfg.UseSSL {
		return routeros.DialContext(ctx, addr, username, password)
	}

	tlsCfg := &tls.Config{ServerName: c.cfg.Hostname}

	if c.cfg.NoSSLCertificate || !c.cfg.SSLCertificateVerify || !c.cfg.SSLCheckHostname {
		insecureReasons := []string{}
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

	return routeros.DialTLSContext(ctx, addr, username, password, tlsCfg)
}

func (c *Connection) RunStream(ctx context.Context, callback func(map[string]string), sentence ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return fmt.Errorf("routers: not connected to %s@%s", c.cfg.RouterName, c.cfg.Hostname)
	}

	cl := c.client
	// Increase buffer slightly to ensure smooth streaming
	cl.Queue = 1000

	// ListenArgs handles the stream safely without destroying the underlying socket on context cancel
	l, err := cl.ListenArgs(sentence)
	if err != nil {
		if c.client == cl {
			_ = cl.Close()
			c.client = nil
			c.recordFailure(time.Now())
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
				// Stream finished or connection dropped
				if err := l.Err(); err != nil {
					if c.client == cl {
						_ = cl.Close()
						c.client = nil
						c.recordFailure(time.Now())
					}
					return err
				}
				return ctx.Err()
			}

			// Process the record if we haven't timed out
			if ctxChan != nil && sen != nil && len(sen.Map) > 0 {
				callback(sen.Map)
			}
		}
	}
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
		if slog.Default().Enabled(ctx, slog.LevelDebug) {
			slog.Debug("In connect timeout",
				"name", c.cfg.RouterName,
				"remaining_secs", int(remaining.Seconds()),
				"failures", c.failureCount)
		}
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
