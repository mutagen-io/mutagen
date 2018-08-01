package daemon

import (
	"time"
)

const (
	// RecommendedDialTimeout is the recommended timeout to use when connecting
	// to the daemon over IPC.
	RecommendedDialTimeout = 1 * time.Second
)
