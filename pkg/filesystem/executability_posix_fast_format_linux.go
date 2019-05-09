package filesystem

// probeExecutabilityPreservationFastByFormat checks if the specified format
// matches well-known executability preservation behavior.
func probeExecutabilityPreservationFastByFormat(f Format) (bool, bool) {
	switch f {
	case FormatEXT:
		return true, true
	case FormatNFS:
		return true, true
	default:
		return false, false
	}
}
