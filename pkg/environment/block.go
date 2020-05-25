package environment

import (
	"strings"
)

// ParseBlock parses an environment variable block of the form
// VAR1=value1[\r]\nVAR2=value2[\r]\n... into a slice of KEY=value strings. It
// opts for performance over extensive format validation.
func ParseBlock(block string) []string {
	// Replace all instances of \r\n with \n.
	block = strings.ReplaceAll(block, "\r\n", "\n")

	// Trim whitespace from around the block.
	block = strings.TrimSpace(block)

	// Split the block into individual lines.
	return strings.Split(block, "\n")
}
