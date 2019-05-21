package numeric

const (
	// MaxUint64 is the maximum value that can be stored in a 64-bit unsigned
	// integer. We define it here to avoid a dependency on the (rather large)
	// math package, but we test to ensure it's equivalent to that package's
	// definition.
	MaxUint64 = 1<<64 - 1

	// MaxUint64Description is a human-friendly mathematic description of
	// MaxUint64.
	MaxUint64Description = "2⁶⁴−1"
)
