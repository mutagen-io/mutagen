package synchronization

import (
	"github.com/mutagen-io/mutagen/pkg/url"
)

// URL represents synchronization endpoint URL.
type URL struct {
	// Protocol endpoint transport protocol.
	Protocol url.Protocol `json:"protocol"`
	// User is the endpoint user.
	User string `json:"user,omitempty"`
	// Host is the endpoint host.
	Host string `json:"host,omitempty"`
	// Port is the endpoint port.
	Port uint16 `json:"port,omitempty"`
	// Path is the synchronization root on the endpoint.
	Path string `json:"path,omitempty"`
	// Environment is the environment variable map to use for the transport.
	Environment map[string]string `json:"environment,omitempty"`
	// Parameters is the parameter map to use for the transport.
	Parameters map[string]string `json:"parameters,omitempty"`
}

// NewURLFromInternalURL creates a new URL representation from an internal
// Protocol Buffers representation. The URL must be a valid synchronization URL.
func NewURLFromInternalURL(url *url.URL) *URL {
	return &URL{
		Protocol:    url.Protocol,
		User:        url.User,
		Host:        url.Host,
		Port:        uint16(url.Port),
		Path:        url.Path,
		Environment: url.Environment,
		Parameters:  url.Parameters,
	}
}
