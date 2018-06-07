package session

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/configuration"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/sync"
)

func (c *Configuration) EnsureValid() error {
	// A nil configuration is not considered valid.
	if c == nil {
		return errors.New("nil configuration")
	}

	// Verify that all ignores specifications in the session are valid.
	for _, ignore := range c.Ignores {
		if !sync.ValidIgnorePattern(ignore) {
			return errors.Errorf("invalid ignore pattern: %s", ignore)
		}
	}

	// Verify that the symlink mode is unspecified or supported for usage.
	if c.SymlinkMode != sync.SymlinkMode_Default && !c.SymlinkMode.Supported() {
		return errors.New("unknown or unsupported symlink mode")
	}

	// Verify that the watch mode is unspecified or supported for usage.
	if c.WatchMode != filesystem.WatchMode_Default && !c.WatchMode.Supported() {
		return errors.New("unknown or unsupported watch mode")
	}

	// Success.
	return nil
}

// snapshotGlobalConfiguration loads the global configuration, transfers the
// relevant parameters to a session configuration, and returns the resulting
// value. It does not validate the
func snapshotGlobalConfiguration() (*Configuration, error) {
	// Load the global configuration.
	configuration, err := configuration.Load()
	if err != nil {
		return nil, errors.Wrap(err, "unable to load global configuration")
	}

	// Create a session configuration object.
	result := &Configuration{}

	// Propagate default ignores.
	result.Ignores = configuration.Ignore.Default

	// Propagate symlink mode.
	if configuration.Symlink.Mode != "" {
		if m, err := sync.NewSymlinkModeFromString(configuration.Symlink.Mode); err != nil {
			return nil, errors.Wrap(err, "unable to parse symlink mode")
		} else {
			result.SymlinkMode = m
		}
	}

	// Propagate watch mode.
	if configuration.Watch.Mode != "" {
		if m, err := filesystem.NewWatchModeFromString(configuration.Watch.Mode); err != nil {
			return nil, errors.Wrap(err, "unable to parse watch mode")
		} else {
			result.WatchMode = m
		}
	}

	// Propagate polling interval.
	result.WatchPollingInterval = configuration.Watch.PollingInterval

	// Verify that the resulting configuration is valid.
	if err := result.EnsureValid(); err != nil {
		return nil, errors.Wrap(err, "global configuration invalid")
	}

	// Success.
	return result, nil
}

// MergeConfigurations merges a per-session and global configuration, allowing
// the per-session configuration to merge with or override the global
// configuration.
func MergeConfigurations(session, global *Configuration) *Configuration {
	// Create the resulting configuration.
	result := &Configuration{}

	// Merge ignores.
	result.Ignores = append(result.Ignores, global.Ignores...)
	result.Ignores = append(result.Ignores, session.Ignores...)

	// Merge symlink mode.
	if session.SymlinkMode != sync.SymlinkMode_Default {
		result.SymlinkMode = session.SymlinkMode
	} else {
		result.SymlinkMode = global.SymlinkMode
	}

	// Merge watch mode.
	if session.WatchMode != filesystem.WatchMode_Default {
		result.WatchMode = session.WatchMode
	} else {
		result.WatchMode = global.WatchMode
	}

	// Merge polling interval.
	if session.WatchPollingInterval != 0 {
		result.WatchPollingInterval = session.WatchPollingInterval
	} else {
		result.WatchPollingInterval = global.WatchPollingInterval
	}

	// Done.
	return result
}
