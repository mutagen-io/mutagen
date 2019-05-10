package format

// Format represents a filesystem volume format.
type Format uint8

const (
	// FormatUnknown represents an unknown volume format. It is supported on all
	// platforms.
	FormatUnknown Format = iota
)
