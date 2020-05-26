package environment

import (
	"strings"
)

// ToMap converts an environment variable specification from a slice of
// "KEY=value" strings to a map with equivalent contents. Any entries not
// adhering to the specified format are ignored. Entries are processed in order,
// meaning that the last entry seen for a key will be what populates the map.
func ToMap(environment []string) map[string]string {
	// Allocate result storage.
	result := make(map[string]string, len(environment))

	// Convert variables.
	for _, specification := range environment {
		keyValue := strings.SplitN(specification, "=", 2)
		if len(keyValue) != 2 {
			continue
		}
		result[keyValue[0]] = keyValue[1]
	}

	// Done.
	return result
}

// FromMap converts a map of environment variables into a slice of "KEY=value"
// strings. If the provided environment is nil, then the resulting slice will be
// nil. If the provided environment is non-nil but empty, then the resulting
// slice will be empty. These two properties are critical to usage with the
// os/exec package.
func FromMap(environment map[string]string) []string {
	// If the environment is nil, then return a nil slice.
	if environment == nil {
		return nil
	}

	// Allocate result storage.
	result := make([]string, 0, len(environment))

	// Convert entries.
	for key, value := range environment {
		result = append(result, key+"="+value)
	}

	// Done.
	return result
}
