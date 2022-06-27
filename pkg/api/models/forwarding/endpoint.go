package forwarding

import (
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/url"
)

// Endpoint represents a forwarding endpoint.
type Endpoint struct {
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
	// Configuration is the endpoint-specific configuration.
	Configuration
	// Connected indicates whether or not the controller is currently connected
	// to the endpoint.
	Connected bool `json:"connected"`
	// EndpointState stores state fields relevant to connected endpoints. It is
	// non-nil if and only if the endpoint is connected.
	*EndpointState
}

// EndpointState encodes the current state of a forwarding endpoint.
type EndpointState struct{}

// loadFromInternal sets an Endpoint to match internal Protocol Buffers
// representations. All parameters must be valid.
func (e *Endpoint) loadFromInternal(url *url.URL, configuration *forwarding.Configuration, state *forwarding.EndpointState) {
	// Propagate URL parameters.
	e.Protocol = url.Protocol
	e.User = url.User
	e.Host = url.Host
	e.Port = uint16(url.Port)
	e.Endpoint = url.Path
	e.Environment = url.Environment
	e.Parameters = url.Parameters

	// Propagate configuration.
	e.Configuration.loadFromInternal(configuration)

	// Propagate connectivity.
	e.Connected = state.Connected

	// Propagate other state fields.
	if !e.Connected {
		e.EndpointState = nil
	} else {
		e.EndpointState = &EndpointState{}
	}
}
