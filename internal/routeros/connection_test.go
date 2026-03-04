package routeros

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConnectDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		failureCount int
		backoff      BackoffConfig
		want         time.Duration
	}{
		{
			name:         "no_failures_returns_initial",
			failureCount: 0,
			backoff:      DefaultBackoff,
			want:         120 * time.Second,
		},
		{
			name:         "five_failures_doubles",
			failureCount: 5,
			backoff:      DefaultBackoff,
			want:         240 * time.Second, // 120 * (1 + 5/5)
		},
		{
			name:         "ten_failures_triples",
			failureCount: 10,
			backoff:      DefaultBackoff,
			want:         360 * time.Second, // 120 * (1 + 10/5)
		},
		{
			name:         "capped_at_max",
			failureCount: 50,
			backoff:      DefaultBackoff,
			want:         900 * time.Second, // would be 1320s, capped
		},
		{
			name:         "zero_divisor_falls_back_to_default",
			failureCount: 0,
			backoff:      BackoffConfig{Divisor: 0},
			want:         120 * time.Second,
		},
		{
			name:         "custom_backoff",
			failureCount: 2,
			backoff:      BackoffConfig{InitialDelay: 10 * time.Second, MaxDelay: 60 * time.Second, Divisor: 2},
			want:         20 * time.Second, // 10 * (1 + 2/2)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Connection{cfg: ConnectionConfig{Backoff: tt.backoff}}
			c.failureCount = tt.failureCount
			got := c.connectDelay()
			if got != tt.want {
				t.Errorf("connectDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInBackoff(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("no_failures_not_in_backoff", func(t *testing.T) {
		t.Parallel()
		c := &Connection{cfg: ConnectionConfig{Backoff: DefaultBackoff}}
		if c.inBackoff(ctx, time.Now()) {
			t.Error("should not be in backoff with no failures recorded")
		}
	})

	t.Run("recent_failure_in_backoff", func(t *testing.T) {
		t.Parallel()
		c := &Connection{cfg: ConnectionConfig{Backoff: DefaultBackoff}}
		c.failureCount = 1
		c.lastFailure = time.Now()
		if !c.inBackoff(ctx, time.Now()) {
			t.Error("should be in backoff immediately after failure")
		}
	})

	t.Run("expired_backoff_not_in_backoff", func(t *testing.T) {
		t.Parallel()
		c := &Connection{cfg: ConnectionConfig{Backoff: DefaultBackoff}}
		c.failureCount = 1
		// Set last failure well beyond the max delay.
		c.lastFailure = time.Now().Add(-2000 * time.Second)
		if c.inBackoff(ctx, time.Now()) {
			t.Error("should not be in backoff after delay expired")
		}
	})
}

func TestRecordFailure(t *testing.T) {
	t.Parallel()

	c := &Connection{cfg: ConnectionConfig{Backoff: DefaultBackoff}}
	now := time.Now()

	c.recordFailure(now)

	if c.failureCount != 1 {
		t.Errorf("failureCount = %d, want 1", c.failureCount)
	}
	if !c.lastFailure.Equal(now) {
		t.Errorf("lastFailure = %v, want %v", c.lastFailure, now)
	}

	c.recordFailure(now)
	if c.failureCount != 2 {
		t.Errorf("failureCount = %d, want 2 after second failure", c.failureCount)
	}
}

func TestResolveCredentials_Direct(t *testing.T) {
	t.Parallel()

	c := &Connection{cfg: ConnectionConfig{Username: "admin", Password: "secret"}}
	u, p, err := c.resolveCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u != "admin" {
		t.Errorf("username = %q, want admin", u)
	}
	if p != "secret" {
		t.Errorf("password = %q, want secret", p)
	}
}

func TestResolveCredentials_FromFile(t *testing.T) {
	t.Parallel()

	credFile := filepath.Join(t.TempDir(), "creds.yaml")
	if err := os.WriteFile(credFile, []byte("username: fileuser\npassword: filepass\n"), 0o600); err != nil {
		t.Fatalf("write creds file: %v", err)
	}

	c := &Connection{cfg: ConnectionConfig{
		Username:        "fallback",
		Password:        "fallback",
		CredentialsFile: credFile,
	}}

	u, p, err := c.resolveCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u != "fileuser" {
		t.Errorf("username = %q, want fileuser", u)
	}
	if p != "filepass" {
		t.Errorf("password = %q, want filepass", p)
	}
}

func TestResolveCredentials_FromFile_PartialOverride(t *testing.T) {
	t.Parallel()

	credFile := filepath.Join(t.TempDir(), "creds.yaml")
	// Only password in file — username comes from config.
	if err := os.WriteFile(credFile, []byte("password: newpass\n"), 0o600); err != nil {
		t.Fatalf("write creds file: %v", err)
	}

	c := &Connection{cfg: ConnectionConfig{
		Username:        "cfguser",
		Password:        "cfgpass",
		CredentialsFile: credFile,
	}}

	u, p, err := c.resolveCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u != "cfguser" {
		t.Errorf("username = %q, want cfguser (fallback from config)", u)
	}
	if p != "newpass" {
		t.Errorf("password = %q, want newpass (from file)", p)
	}
}

func TestResolveCredentials_MissingFile(t *testing.T) {
	t.Parallel()

	c := &Connection{cfg: ConnectionConfig{
		CredentialsFile: "/nonexistent/path/creds.yaml",
	}}

	_, _, err := c.resolveCredentials()
	if err == nil {
		t.Error("expected error for missing credentials file")
	}
}

func TestResolveCredentials_MalformedFile(t *testing.T) {
	t.Parallel()

	credFile := filepath.Join(t.TempDir(), "creds.yaml")
	if err := os.WriteFile(credFile, []byte("not: valid: yaml: [\n"), 0o600); err != nil {
		t.Fatalf("write creds file: %v", err)
	}

	c := &Connection{cfg: ConnectionConfig{CredentialsFile: credFile}}
	_, _, err := c.resolveCredentials()
	if err == nil {
		t.Error("expected error for malformed YAML credentials file")
	}
}

func TestIsConnected_InitiallyFalse(t *testing.T) {
	t.Parallel()

	c := NewConnection(ConnectionConfig{RouterName: "test", Hostname: "localhost"})
	if c.IsConnected() {
		t.Error("new connection should not be connected")
	}
}

func TestRouterName(t *testing.T) {
	t.Parallel()

	c := NewConnection(ConnectionConfig{RouterName: "my-router"})
	if c.RouterName() != "my-router" {
		t.Errorf("RouterName() = %q, want my-router", c.RouterName())
	}
}
