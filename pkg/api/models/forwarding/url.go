package forwarding

import (
	"github.com/mutagen-io/mutagen/pkg/url"
)

// URL represents forwarding endpoint URL.
type URL struct {
	// Protocol endpoint transport protocol.
	Protocol url.Protocol `json:"protocol"`
	// User is the endpoint user.
	User string `json:"user,omitempty"`
	// Host is the endpoint host.
	Host string `json:"host,omitempty"`
	// Port is the endpoint port.
	Port uint16 `json:"port,omitempty"`
	// Endpoint is the listening or dialing address on the endpoint.
	Endpoint string `json:"endpoint,omitempty"`
	// Environment is the environment variable map to use for the transport.
	Environment map[string]string `json:"environment,omitempty"`
	// Parameters is the parameter map to use for the transport.
	Parameters map[string]string `json:"parameters,omitempty"`
}

// LoadFromInternalURL sets a URL to match an internal Protocol Buffers
// representation. The URL must be a valid forwarding URL.
func (u *URL) LoadFromInternalURL(url *url.URL) {
	u.Protocol = url.Protocol
	u.User = url.User
	u.Host = url.Host
	u.Port = uint16(url.Port)
	u.Endpoint = url.Path
	u.Environment = url.Environment
	u.Parameters = url.Parameters
}
