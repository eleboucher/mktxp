package collector

import "context"

// BackgroundTasker defines an optional interface for collectors that need background tasks.
type BackgroundTasker interface {
	StartBackgroundTest(ctx context.Context, collectorName string)
}
