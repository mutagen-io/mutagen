package filesystem

// probeExecutabilityPreservationFastByFormat checks if the specified format
// matches well-known executability preservation behavior.
func probeExecutabilityPreservationFastByFormat(f Format) (bool, bool) {
	switch f {
	case FormatAPFS:
		return true, true
	case FormatHFS:
		return true, true
	default:
		return false, false
	}
}
