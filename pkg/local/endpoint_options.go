package local

import (
	"context"

	fs "github.com/havoc-io/mutagen/pkg/filesystem"
)

// endpointOptions controls the override behavior for a local endpoint.
type endpointOptions struct {
	// maximumEntryCountCallback can specify a callback that will return an
	// override value for the maximum entry count.
	maximumEntryCountCallback func() uint64
	// maximumStagingFileSizeCallback can specify a callback that will return an
	// override value for the maximum stagin file size.
	maximumStagingFileSizeCallback func() uint64
	// cachePathCallback can specify a callback that will be used to compute the
	// cache path.
	cachePathCallback func(string, bool) (string, error)
	// stagingRootCallback can specify a callback that will be used to compute
	// the staging root.
	stagingRootCallback func(string, bool) (string, error)
	// watchingMechanism can specify a callback that will be used as an
	// alternative mechanism for filesystem watching.
	watchingMechanism func(context.Context, string, chan<- struct{})
	// defaultFileModeCallback can specify a callback that will return an
	// override value for the default file mode.
	defaultFileModeCallback func() fs.Mode
	// defaultFileModeCallback can specify a callback that will return an
	// override value for the default directory mode.
	defaultDirectoryModeCallback func() fs.Mode
	// defaultOwnerUserCallback can specify a callback that will return an
	// override value for the default owner user specification.
	defaultOwnerUserCallback func() string
	// defaultOwnerGroupCallback can specify a callback that will return an
	// override value for the default owner group specification.
	defaultOwnerGroupCallback func() string
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

// WithMaximumEntryCount specifies that the endpoint should use the specified
// maximum entry count instead of what's received from the controller in the
// session configuration.
func WithMaximumEntryCount(count uint64) EndpointOption {
	return newFunctionEndpointOption(func(options *endpointOptions) {
		options.maximumEntryCountCallback = func() uint64 {
			return count
		}
	})
}

// WithMaximumStagingFileSize specifies that the endpoint should use the
// specified maximum staging file size instead of what's received from the
// controller in the session configuration.
func WithMaximumStagingFileSize(size uint64) EndpointOption {
	return newFunctionEndpointOption(func(options *endpointOptions) {
		options.maximumStagingFileSizeCallback = func() uint64 {
			return size
		}
	})
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
// two arguments. The path may exist, but if it does must be a directory.
func WithStagingRootCallback(callback func(string, bool) (string, error)) EndpointOption {
	return newFunctionEndpointOption(func(options *endpointOptions) {
		options.stagingRootCallback = callback
	})
}

// WithWatchingMechanism overrides the filesystem watching function that the
// endpoint uses to monitor for filesystem changes. The specified function will
// be provided with three arguments: a context to indicate watch cancellation,
// the path to be watched (recursively), and an events channel that should be
// populated in a non-blocking fashion every time an event occurs. If an error
// occurs during watching, the event channel should be closed. It should also be
// closed on cancellation.
func WithWatchingMechanism(callback func(context.Context, string, chan<- struct{})) EndpointOption {
	return newFunctionEndpointOption(func(options *endpointOptions) {
		options.watchingMechanism = callback
	})
}

// WithDefaultFileMode specifies that the endpoint should use the specified
// default file mode instead of what's received from the controller in the
// session configuration.
func WithDefaultFileMode(mode fs.Mode) EndpointOption {
	return newFunctionEndpointOption(func(options *endpointOptions) {
		options.defaultFileModeCallback = func() fs.Mode {
			return mode
		}
	})
}

// WithDefaultDirectoryMode specifies that the endpoint should use the specified
// default directory mode instead of what's received from the controller in the
// session configuration.
func WithDefaultDirectoryMode(mode fs.Mode) EndpointOption {
	return newFunctionEndpointOption(func(options *endpointOptions) {
		options.defaultDirectoryModeCallback = func() fs.Mode {
			return mode
		}
	})
}

// WithDefaultOwnerUser specifies that the endpoint should use the specified
// default owner user instead of what's received from the controller in the
// session configuration.
func WithDefaultOwnerUser(user string) EndpointOption {
	return newFunctionEndpointOption(func(options *endpointOptions) {
		options.defaultOwnerUserCallback = func() string {
			return user
		}
	})
}

// WithDefaultOwnerGroup specifies that the endpoint should use the specified
// default owner group instead of what's received from the controller in the
// session configuration.
func WithDefaultOwnerGroup(group string) EndpointOption {
	return newFunctionEndpointOption(func(options *endpointOptions) {
		options.defaultOwnerGroupCallback = func() string {
			return group
		}
	})
}
