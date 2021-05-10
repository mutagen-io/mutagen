package api

import (
	"time"
)

const (
	// ReadTimeout is the read timeout for HTTP requests.
	ReadTimeout = 5 * time.Second
	// IdleTimeout is the connection timeout for idle connections.
	IdleTimeout = 2 * time.Minute
)
