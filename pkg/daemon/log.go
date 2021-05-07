package daemon

import (
	"fmt"
	"io"
	"os"
)

// OpenLog opens the daemon log for writing. The caller is responsible for
// closing the log.
func OpenLog() (io.WriteCloser, error) {
	// Compute the log file path.
	path, err := logPath()
	if err != nil {
		return nil, fmt.Errorf("unable to determine daemon log path: %w", err)
	}

	// Open the log.
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
}
