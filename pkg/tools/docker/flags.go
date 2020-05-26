package docker

import (
	"errors"
	"fmt"
)

// DaemonConnectionFlags encodes top-level Docker command line flags that
// control the Docker daemon connection. These flags are shared between the
// Docker CLI and Docker Compose. These flags can be loaded from Mutagen URL
// parameters or used as command line flag storage. The zero value of this
// structure is a valid value corresponding to the absence of any of these
// flags.
type DaemonConnectionFlags struct {
	// Host stores the value of the -H/--host flag.
	Host string
	// Context stores the value of the -c/--context flag.
	Context string
	// TLS indicates the presence of the --tls flag.
	TLS bool
	// TLSCACert stores the value of the --tlscacert flag.
	TLSCACert string
	// TLSCert stores the value of the --tlscert flag.
	TLSCert string
	// TLSKey stores the value of the --tlskey flag.
	TLSKey string
	// TLSVerify indicates the presence of the --tlsverify flag.
	TLSVerify bool
}

// LoadDaemonConnectionFlagsFromURLParameters loads top-level Docker daemon
// connection flags from Mutagen URL parameters.
func LoadDaemonConnectionFlagsFromURLParameters(parameters map[string]string) (*DaemonConnectionFlags, error) {
	// Create a zero-valued result (corresponding to no flags).
	result := &DaemonConnectionFlags{}

	// Validate and convert parameters.
	for key, value := range parameters {
		switch key {
		case "context":
			if value == "" {
				return nil, errors.New("context parameter has empty value")
			}
			result.Context = value
		case "host":
			if value == "" {
				return nil, errors.New("host parameter has empty value")
			}
			result.Host = value
		case "tls":
			if value != "" {
				return nil, errors.New("tls parameter has non-empty value")
			}
			result.TLS = true
		case "tlscacert":
			if value == "" {
				return nil, errors.New("tlacacert parameter has empty value")
			}
			result.TLSCACert = value
		case "tlscert":
			if value == "" {
				return nil, errors.New("tlscert parameter has empty value")
			}
			result.TLSCert = value
		case "tlskey":
			if value == "" {
				return nil, errors.New("tlskey parameter has empty value")
			}
			result.TLSKey = value
		case "tlsverify":
			if value != "" {
				return nil, errors.New("tlsverify parameter has non-empty value")
			}
			result.TLSVerify = true
		default:
			return nil, fmt.Errorf("unknown parameter: %s", key)
		}
	}

	// Success.
	return result, nil
}

// ToFlags reconstitues top-level daemon connection flags so that they can be
// passed to a Docker CLI or Docker Compose command.
func (f *DaemonConnectionFlags) ToFlags() []string {
	// Set up the result.
	var result []string

	// Add flags as necessary.
	if f.Host != "" {
		result = append(result, "--host", f.Host)
	}
	if f.Context != "" {
		result = append(result, "--context", f.Context)
	}
	if f.TLS {
		result = append(result, "--tls")
	}
	if f.TLSCACert != "" {
		result = append(result, "--tlscacert", f.TLSCACert)
	}
	if f.TLSCert != "" {
		result = append(result, "--tlscert", f.TLSCert)
	}
	if f.TLSKey != "" {
		result = append(result, "--tlskey", f.TLSKey)
	}
	if f.TLSVerify {
		result = append(result, "--tlsverify")
	}

	// Done.
	return result
}

// ToURLParameters converts top-level daemon connection flags to parameters that
// can be embedded in a Mutagen URL. These parameters can be converted back
// using LoadDaemonConnectionFlagsFromURLParameters.
func (f *DaemonConnectionFlags) ToURLParameters() map[string]string {
	// Create an empty set of parameters.
	result := make(map[string]string)

	// Add parameters as necessary.
	if f.Host != "" {
		result["host"] = f.Host
	}
	if f.Context != "" {
		result["context"] = f.Context
	}
	if f.TLS {
		result["tls"] = ""
	}
	if f.TLSCACert != "" {
		result["tlscacert"] = f.TLSCACert
	}
	if f.TLSCert != "" {
		result["tlscert"] = f.TLSCert
	}
	if f.TLSKey != "" {
		result["tlskey"] = f.TLSKey
	}
	if f.TLSVerify {
		result["tlsverify"] = ""
	}

	// Done.
	return result
}
