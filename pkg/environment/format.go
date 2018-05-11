package environment

import (
	"fmt"
)

func Format(environment map[string]string) []string {
	// Create the result.
	result := make([]string, 0, len(environment))

	// Add entries.
	for k, v := range environment {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	// Success.
	return result
}
