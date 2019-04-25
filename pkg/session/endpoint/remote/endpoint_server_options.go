package remote

import (
	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/session/endpoint/local"
)

// EndpointConnectionValidator is a validator function type for validating
// endpoint server connections. It is passed the session root, identifier,
// version, configuration, and role information provided by the connection. If
// it returns a non-nil error, the connection will be terminated before serving
// begins. The validator should not mutate the provided configuration. The
// provided root and configuration passed to the validation function will take
// into account any overrides specified by endpoint server options.
type EndpointConnectionValidator func(string, string, session.Version, *session.Configuration, bool) error

// endpointServerOptions controls the override behavior for an endpoint server.
type endpointServerOptions struct {
	// root specifies an override for the session root path.
	root string
	// configuration specifies endpoint-specific configuration overrides.
	configuration *session.Configuration
	// connectionValidator specifies a validation function for the received
	// endpoint configuration (with root and configuration overrides taken into
	// account).
	connectionValidator EndpointConnectionValidator
	// endpointOptions is a collection of endpoint options that will be passed
	// to the underlying endpoint.
	endpointOptions []local.EndpointOption
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

// WithConfiguration allows for overriding certain endpoint-specific parameters.
// The provided Configuration object will be validated to ensure that that it
// only overrides parameters which are valid to override on an endpoint-specific
// basis.
func WithConfiguration(configuration *session.Configuration) EndpointServerOption {
	return newFunctionEndpointServerOption(func(options *endpointServerOptions) {
		options.configuration = configuration
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
