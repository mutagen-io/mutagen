package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// loadEnvironment computes the effective set of environment variables using the
// same rules as Docker Compose.
func loadEnvironment(projectDirectory, environmentFileName string) (map[string]string, error) {
	// Create an empty (but initialized) environment.
	environment := make(map[string]string)

	// Load the environment file (if it exists) and add its contents.
	environmentFilePath := filepath.Join(projectDirectory, environmentFileName)
	fileEnvironment, err := godotenv.Read(environmentFilePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to load environment file (%s): %w", environmentFilePath, err)
	}
	for key, value := range fileEnvironment {
		environment[key] = value
	}

	// Add environment variables from the OS.
	for _, specification := range os.Environ() {
		keyValue := strings.SplitN(specification, "=", 2)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("invalid OS environment variable specification: %s", specification)
		}
		environment[keyValue[0]] = keyValue[1]
	}

	// Success.
	return environment, nil
}
