package ignore

import (
	"fmt"
)

// IsDefault indicates whether or not the ignore syntax is
// IgnoreSyntax_IgnoreSyntaxDefault.
func (s IgnoreSyntax) IsDefault() bool {
	return s == IgnoreSyntax_IgnoreSyntaxDefault
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (s IgnoreSyntax) MarshalText() ([]byte, error) {
	var result string
	switch s {
	case IgnoreSyntax_IgnoreSyntaxDefault:
	case IgnoreSyntax_IgnoreSyntaxGit:
		result = "git"
	case IgnoreSyntax_IgnoreSyntaxDocker:
		result = "docker"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
func (s *IgnoreSyntax) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to an ignore syntax.
	switch text {
	case "git":
		*s = IgnoreSyntax_IgnoreSyntaxGit
	case "docker":
		*s = IgnoreSyntax_IgnoreSyntaxDocker
	default:
		return fmt.Errorf("unknown ignore syntax specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular ignore syntax is a valid,
// non-default value.
func (s IgnoreSyntax) Supported() bool {
	switch s {
	case IgnoreSyntax_IgnoreSyntaxGit:
		return true
	case IgnoreSyntax_IgnoreSyntaxDocker:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of an ignore syntax.
func (s IgnoreSyntax) Description() string {
	switch s {
	case IgnoreSyntax_IgnoreSyntaxDefault:
		return "Default"
	case IgnoreSyntax_IgnoreSyntaxGit:
		return "Git"
	case IgnoreSyntax_IgnoreSyntaxDocker:
		return "Docker"
	default:
		return "Unknown"
	}
}
