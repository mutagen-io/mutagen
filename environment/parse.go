package environment

import (
	"strings"

	"github.com/pkg/errors"
)

func Parse(environment []string) (map[string]string, error) {
	// Create the result.
	result := make(map[string]string, len(environment))

	// Process each line.
	for _, e := range environment {
		components := strings.SplitN(e, "=", 2)
		if len(components) != 2 {
			return nil, errors.Errorf("invalid variable specification: %s", e)
		}
		result[components[0]] = components[1]
	}

	// Success.
	return result, nil
}

// TODO: When documenting this function, make a note that it's designed to be
// platform-agnostic, since we use it on remotes as well, and that's why it does
// the newline replacement.
func ParseBlock(environment string) (map[string]string, error) {
	// Convert line endings, trim trailing newlines, and split the output into
	// individual lines.
	environment = strings.Replace(environment, "\r\n", "\n", -1)
	environment = strings.TrimSpace(environment)
	lines := strings.Split(environment, "\n")

	// Call the base parse function.
	return Parse(lines)
}
