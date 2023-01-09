package stream

// Flusher represents a stream that performs internal buffering that may need to
// be flushed to ensure transmission.
type Flusher interface {
	// Flush forces transmission of any buffered stream data.
	Flush() error
}
