package forwarding

// IsValidProtocol returns whether or not the specified protocol is valid for
// use in forwarding (either as a from or to address).
func IsValidProtocol(protocol string) bool {
	switch protocol {
	case "tcp":
		return true
	case "tcp4":
		return true
	case "tcp6":
		return true
	case "unix":
		return true
	case "npipe":
		return true
	default:
		return false
	}
}
