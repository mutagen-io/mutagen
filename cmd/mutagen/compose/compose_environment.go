package compose

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// loadEnvironment loads a "dotenv" environment variable file from disk and
// updates it to include variables from the current process' environment (with
// the current process' environment taking precedence). If the target file
// doesn't exist, then it is treated as empty and the resulting environment will
// be the current process' environment.
func loadEnvironment(path string) (map[string]string, error) {
	// Create an empty (but initialized) environment.
	environment := make(map[string]string)

	// Load the environment file (if it exists) and add its contents. It's worth
	// noting that the godotenv package supports interpolation by default, which
	// is what Docker Compose uses by default when loading environment variable
	// files from disk.
	fileEnvironment, err := godotenv.Read(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to load environment file (%s): %w", path, err)
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
