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
// strings.
func FromMap(environment map[string]string) []string {
	// Allocate result storage.
	result := make([]string, 0, len(environment))

	// Convert entries.
	for key, value := range environment {
		result = append(result, key+"="+value)
	}

	// Done.
	return result
}
