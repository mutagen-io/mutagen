package compose

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// LoadEnvironment loads a "dotenv" environment variable file from disk and
// updates it to include variables from the current process' environment (with
// the current process' environment taking precedence). Interpolation is enabled
// by default for the contents of the environment file. If the target file
// doesn't exist, then it is treated as empty and the resulting environment will
// be the current process' environment.
func LoadEnvironment(path string) (map[string]string, error) {
	// Load the environment file (if it exists) and add its contents. It's worth
	// noting that the godotenv package supports interpolation by default, which
	// is what Docker Compose uses by default when loading environment variable
	// files from disk.
	environment, err := godotenv.Read(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to load environment file (%s): %w", path, err)
	}

	// Grab the environment from the OS.
	osEnvironment := os.Environ()

	// If the environment wasn't allocated, then do so now.
	if environment == nil {
		environment = make(map[string]string, len(osEnvironment))
	}

	// Add environment variables from the OS.
	for _, specification := range osEnvironment {
		keyValue := strings.SplitN(specification, "=", 2)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("invalid OS environment variable specification: %s", specification)
		}
		environment[keyValue[0]] = keyValue[1]
	}

	// Success.
	return environment, nil
}
