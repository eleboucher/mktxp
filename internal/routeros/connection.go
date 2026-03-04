package routeros

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	gorouteros "github.com/go-routeros/routeros/v3"
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
	client       *gorouteros.Client
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
		return fmt.Errorf("routeros: connect %s@%s: %w", c.cfg.RouterName, c.cfg.Hostname, err)
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
		return nil, fmt.Errorf("routeros: not connected to %s@%s", c.cfg.RouterName, c.cfg.Hostname)
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

func (c *Connection) dial(ctx context.Context, username, password string) (*gorouteros.Client, error) {
	addr := fmt.Sprintf("%s:%d", c.cfg.Hostname, c.cfg.Port)
	timeout := c.cfg.SocketTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if !c.cfg.UseSSL {
		return gorouteros.DialContext(ctx, addr, username, password)
	}

	slog.Warn("Connecting with TLS", "host", c.cfg.Hostname, "insecure", c.cfg.NoSSLCertificate)

	tlsCfg := &tls.Config{ServerName: c.cfg.Hostname}
	if c.cfg.NoSSLCertificate || !c.cfg.SSLCertificateVerify || !c.cfg.SSLCheckHostname {
		tlsCfg.InsecureSkipVerify = true //nolint:gosec
	}

	return gorouteros.DialTLSContext(ctx, addr, username, password, tlsCfg)
}

func (c *Connection) resolveCredentials() (string, string, error) {
	if c.cfg.CredentialsFile == "" {
		return c.cfg.Username, c.cfg.Password, nil
	}
	data, err := os.ReadFile(c.cfg.CredentialsFile)
	if err != nil {
		return "", "", fmt.Errorf("routeros: read credentials file %q: %w", c.cfg.CredentialsFile, err)
	}
	var creds credentialsFile
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return "", "", fmt.Errorf("routeros: parse credentials file %q: %w", c.cfg.CredentialsFile, err)
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
