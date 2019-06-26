package ipc

import (
	"time"
)

const (
	// RecommendedDialTimeout is the recommended timeout to use when
	// establishing IPC connections.
	RecommendedDialTimeout = 1 * time.Second
)
