package compose

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// environmentFromOS converts the result of os.Environ to a map type supported
// by the compose-go package.
func environmentFromOS() map[string]string {
	// Grab the environment variable specification.
	environment := os.Environ()

	// Convert specifications.
	result := make(map[string]string, len(environment))
	for _, specification := range environment {
		keyValue := strings.SplitN(specification, "=", 2)
		if len(keyValue) != 2 {
			panic("invalid environment")
		}
		result[keyValue[0]] = keyValue[1]
	}

	// Done.
	return result
}

// environmentFromFile loads environment variable specifications from a file
// adhering to the Docker Compose environment file format:
// https://docs.docker.com/compose/env-file/
func environmentFromFile(path string) (map[string]string, error) {
	return godotenv.Read(path)
}
