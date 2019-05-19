package local

// endpointOptions controls the override behavior for a local endpoint.
type endpointOptions struct {
	// cachePathCallback can specify a callback that will be used to compute the
	// cache path.
	cachePathCallback func(string, bool) (string, error)
	// stagingRootCallback can specify a callback that will be used to compute
	// the staging root.
	stagingRootCallback func(string, bool) (string, bool, error)
}

// EndpointOption is the interface for specifying endpoint options. It cannot be
// constructed or implemented directly, only by one of option constructors
// provided by this package.
type EndpointOption interface {
	// apply modifies the provided endpoint options configuration in accordance
	// with the option.
	apply(*endpointOptions)
}

// functionEndpointOption is an implementation of EndpointOption that adapts a
// simple closure function to be an EndpointOption.
type functionEndpointOption struct {
	applier func(*endpointOptions)
}

// newFunctionEndpointOption creates a new EndpointOption using a simple closure
// function.
func newFunctionEndpointOption(applier func(*endpointOptions)) EndpointOption {
	return &functionEndpointOption{applier}
}

// apply implements EndpointOption.apply for functionEndpointOption.
func (o *functionEndpointOption) apply(options *endpointOptions) {
	o.applier(options)
}

// WithCachePathCallback overrides the function that the endpoint uses to
// compute cache storage paths. The specified callback will be provided with two
// arguments: the session identifier (a UUID) and a boolean indicating whether
// or not this is the alpha endpoint (if false, it's the beta endpoint). The
// function should return a path that is consistent but unique in terms of these
// two arguments.
func WithCachePathCallback(callback func(string, bool) (string, error)) EndpointOption {
	return newFunctionEndpointOption(func(options *endpointOptions) {
		options.cachePathCallback = callback
	})
}

// WithStagingRootCallback overrides the function that the endpoint uses to
// compute staging root paths. The specified callback will be provided with two
// arguments: the session identifier (a UUID) and a boolean indicating whether
// or not this is the alpha endpoint (if false, it's the beta endpoint). The
// function should return a path that is consistent but unique in terms of these
// two arguments, as well as whether or not the generated staging root should be
// marked as hidden when re-created. The path may exist, but if it does must be
// a directory, and it will be regularly deleted/re-created.
func WithStagingRootCallback(callback func(string, bool) (string, bool, error)) EndpointOption {
	return newFunctionEndpointOption(func(options *endpointOptions) {
		options.stagingRootCallback = callback
	})
}
