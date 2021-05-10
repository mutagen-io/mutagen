package housekeeping

import (
	"context"
	"time"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
)

const (
	// housekeepingInterval is the interval at which housekeeping will be
	// invoked by the agent.
	housekeepingInterval = 24 * time.Hour
)

// HousekeepRegularly provides regular housekeeping operations at a standard
// interval. It is designed to be run as a background Goroutine in a long-lived
// process. It will terminate when the provided context is cancelled.
func HousekeepRegularly(context context.Context, logger *logging.Logger) {
	// Perform an initial housekeeping operation since the ticker won't fire
	// straight away.
	logger.Info("Performing initial housekeeping")
	agent.Housekeep()
	synchronization.Housekeep()

	// Create a ticker to regulate housekeeping and defer its shutdown.
	ticker := time.NewTicker(housekeepingInterval)
	defer ticker.Stop()

	// Loop and wait for the ticker or cancellation.
	for {
		select {
		case <-context.Done():
			return
		case <-ticker.C:
			logger.Info("Performing regular housekeeping")
			agent.Housekeep()
			synchronization.Housekeep()
		}
	}
}
