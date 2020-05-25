package docker

import (
	"errors"
	"fmt"
)

// parametersToTopLevelFlags converts internal Docker URL parameters to their
// corresponding top-level command line flags.
func parametersToTopLevelFlags(parameters map[string]string) ([]string, error) {
	// Set up flag storage.
	var flags []string

	// Validate and convert parameters.
	for key, value := range parameters {
		switch key {
		case "context":
			if value == "" {
				return nil, errors.New("context parameter has empty value")
			}
			flags = append(flags, "--context", value)
		case "host":
			if value == "" {
				return nil, errors.New("host parameter has empty value")
			}
			flags = append(flags, "--host", value)
		case "tls":
			if value != "" {
				return nil, errors.New("tls parameter has non-empty value")
			}
			flags = append(flags, "--tls")
		case "tlscacert":
			if value == "" {
				return nil, errors.New("tlacacert parameter has empty value")
			}
			flags = append(flags, "--tlscacert", value)
		case "tlscert":
			if value == "" {
				return nil, errors.New("tlscert parameter has empty value")
			}
			flags = append(flags, "--tlscert", value)
		case "tlskey":
			if value == "" {
				return nil, errors.New("tlskey parameter has empty value")
			}
			flags = append(flags, "--tlskey", value)
		case "tlsverify":
			if value != "" {
				return nil, errors.New("tlsverify parameter has non-empty value")
			}
			flags = append(flags, "--tlsverify")
		default:
			return nil, fmt.Errorf("unknown parameter: %s", key)
		}
	}

	// Success.
	return flags, nil
}
