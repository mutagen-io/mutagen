package stream

// Flusher represents a stream that performs internal buffering that may need to
// be flushed to ensure transmission.
type Flusher interface {
	// Flush forces transmission of any buffered stream data.
	Flush() error
}

// multiFlusher is the Flusher implementation underlying MultiFlusher.
type multiFlusher struct {
	// flushers are the underlying flushers.
	flushers []Flusher
}

// Flush implements Flusher.Flush.
func (f *multiFlusher) Flush() error {
	for _, flusher := range f.flushers {
		if err := flusher.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// MultiFlusher creates a single flusher that flushes multiple underlying
// flushers. The flushers are flushed in the order specified, and thus higher
// layers should be specified before lower. If an error occurs, then flushing
// halts and subsequent flushers are not flushed.
func MultiFlusher(flushers ...Flusher) Flusher {
	return &multiFlusher{flushers}
}
