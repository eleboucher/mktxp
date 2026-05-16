package collector

import (
	"testing"
	"time"

	"github.com/eleboucher/mktxp/internal/config"
)

func TestBandwidthCollector_StartBackgroundTestRespectsDisabledFlag(t *testing.T) {
	t.Parallel()

	config.Handler.RegisterTestSystemConfig(&config.SystemConfig{
		Bandwidth:             false,
		BandwidthTestInterval: 1,
	})

	c := NewBandwidthCollector()

	done := make(chan struct{})
	go func() {
		c.StartBackgroundTest(t.Context(), "bandwidth")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("StartBackgroundTest did not return immediately when bandwidth is disabled")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.internetResults != nil {
		t.Error("internetResults should remain nil when bandwidth is disabled")
	}
}
