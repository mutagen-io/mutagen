package remote

import (
	"github.com/havoc-io/mutagen/pkg/local"
	"github.com/havoc-io/mutagen/pkg/session"
)

// EndpointConnectionValidator is a validator function type for validating
// endpoint server connections. It is passed the session identifier, session
// version, session configuration, and alpha identification provided by the
// connnection. If it returns a non-nil error, the connection will be terminated
// before serving begins. The validator should not mutate the provided
// configuration.
type EndpointConnectionValidator func(string, session.Version, *session.Configuration, bool) error

// endpointServerOptions controls the override behavior for an endpoint server.
type endpointServerOptions struct {
	root                string
	connectionValidator EndpointConnectionValidator
	endpointOptions     []local.EndpointOption
}

// EndpointServerOption is the interface for specifying endpoint server options.
// It cannot be constructed or implemented directly, only by one of option
// constructors provided by this package.
type EndpointServerOption interface {
	// apply modifies the provided endpoint options configuration in accordance
	// with the option.
	apply(*endpointServerOptions)
}

// functionEndpointServerOption is an implementation of EndpointServerOption
// that adapts a simple closure function to be an EndpointServerOption.
type functionEndpointServerOption struct {
	applier func(*endpointServerOptions)
}

// newFunctionEndpointServerOption creates a new EndpointServerOption using a
// simple closure function.
func newFunctionEndpointServerOption(applier func(*endpointServerOptions)) EndpointServerOption {
	return &functionEndpointServerOption{applier}
}

// apply implements EndpointServerOption.apply for functionEndpointServerOption.
func (o *functionEndpointServerOption) apply(options *endpointServerOptions) {
	o.applier(options)
}

// WithRoot tells the endpoint server to override the incoming root path with
// the specified path. This will be required for custom endpoint servers, where
// the root passed to the server from the client controller will be empty (and
// hence invalid).
func WithRoot(root string) EndpointServerOption {
	return newFunctionEndpointServerOption(func(options *endpointServerOptions) {
		options.root = root
	})
}

// WithConnectionValidator tells the endpoint server to validate the received
// session information with the specified callback. If this validation fails
// (i.e. if the validator returns an error), serving is terminated.
func WithConnectionValidator(validator EndpointConnectionValidator) EndpointServerOption {
	return newFunctionEndpointServerOption(func(options *endpointServerOptions) {
		options.connectionValidator = validator
	})
}

// WithEndpointOption tells the endpoint server to pass the provided endpoint
// option to the underlying endpoint instance.
func WithEndpointOption(option local.EndpointOption) EndpointServerOption {
	return newFunctionEndpointServerOption(func(options *endpointServerOptions) {
		options.endpointOptions = append(options.endpointOptions, option)
	})
}
