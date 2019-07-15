package netpipe

import (
	"github.com/mutagen-io/mutagen/pkg/url"
)

const (
	// Protocol_Netpipe is a fake protocol used to perform integration tests
	// over an in-memory setup of the remote client/server architecture.
	Protocol_Netpipe url.Protocol = -1
)
