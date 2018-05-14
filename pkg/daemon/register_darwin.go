package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/process"
)

const RegistrationSupported = true

const launchdPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>io.mutagen.mutagen</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>daemon</string>
		<string>run</string>
	</array>
	<key>LimitLoadToSessionType</key>
	<string>Aqua</string>
	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`

const (
	libraryDirectoryName        = "Library"
	libraryDirectoryPermissions = 0700

	launchAgentsDirectoryName        = "LaunchAgents"
	launchAgentsDirectoryPermissions = 0755

	launchdPlistName        = "io.mutagen.mutagen.plist"
	launchdPlistPermissions = 0644
)

func Register() error {
	// If we're already registered, don't do anything.
	if registered, err := registered(); err != nil {
		return errors.Wrap(err, "unable to determine registration status")
	} else if registered {
		return nil
	}

	// Acquire the lock to ensure the daemon isn't running. We switch the start
	// and stop mechanism depending on whether or not we're registered, so we
	// need to make sure we don't try to stop a daemon started using a different
	// mechanism.
	lock, err := AcquireLock()
	if err != nil {
		return errors.New("unable to alter registration while daemon is running")
	}
	defer lock.Unlock()

	// Ensure the user's Library directory exists.
	targetPath := filepath.Join(filesystem.HomeDirectory, libraryDirectoryName)
	if err := os.MkdirAll(targetPath, libraryDirectoryPermissions); err != nil {
		return errors.Wrap(err, "unable to create Library directory")
	}

	// Ensure the LaunchAgents directory exists.
	targetPath = filepath.Join(targetPath, launchAgentsDirectoryName)
	if err := os.MkdirAll(targetPath, launchAgentsDirectoryPermissions); err != nil {
		return errors.Wrap(err, "unable to create LaunchAgents directory")
	}

	// Format a launchd plist.
	plist := fmt.Sprintf(launchdPlistTemplate, process.Current.ExecutablePath)

	// Attempt to write the launchd plist.
	targetPath = filepath.Join(targetPath, launchdPlistName)
	if err := filesystem.WriteFileAtomic(targetPath, []byte(plist), launchdPlistPermissions); err != nil {
		return errors.Wrap(err, "unable to write launchd agent plist")
	}

	// Success.
	return nil
}

func Unregister() error {
	// If we're not registered, don't do anything.
	if registered, err := registered(); err != nil {
		return errors.Wrap(err, "unable to determine registration status")
	} else if !registered {
		return nil
	}

	// Acquire the lock to ensure the daemon isn't running. We switch the start
	// and stop mechanism depending on whether or not we're registered, so we
	// need to make sure we don't try to stop a daemon started using a different
	// mechanism.
	lock, err := AcquireLock()
	if err != nil {
		return errors.New("unable to alter registration while daemon is running")
	}
	defer lock.Unlock()

	// Compute the launchd plist path.
	targetPath := filepath.Join(
		filesystem.HomeDirectory,
		libraryDirectoryName,
		launchAgentsDirectoryName,
		launchdPlistName,
	)

	// Attempt to remove the launchd plist.
	if err := os.Remove(targetPath); err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "unable to remove launchd agent plist")
		}
	}

	// Success.
	return nil
}

func registered() (bool, error) {
	// Compute the launchd plist path.
	targetPath := filepath.Join(
		filesystem.HomeDirectory,
		libraryDirectoryName,
		launchAgentsDirectoryName,
		launchdPlistName,
	)

	// Check if it exists and is what's expected.
	if info, err := os.Lstat(targetPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "unable to query launchd agent plist")
	} else if !info.Mode().IsRegular() {
		return false, errors.New("unexpected contents at launchd agent plist path")
	}

	// Success.
	return true, nil
}

func RegisteredStart() (bool, error) {
	// Check if we're registered. If not, we don't handle the start request.
	if registered, err := registered(); err != nil {
		return false, errors.Wrap(err, "unable to determine daemon registration status")
	} else if !registered {
		return false, nil
	}

	// Compute the launchd plist path.
	targetPath := filepath.Join(
		filesystem.HomeDirectory,
		libraryDirectoryName,
		launchAgentsDirectoryName,
		launchdPlistName,
	)

	// Attempt to load the daemon.
	load := exec.Command("launchctl", "load", targetPath)
	if err := load.Run(); err != nil {
		return false, errors.Wrap(err, "unable to load launchd plist")
	}

	// Success.
	return true, nil
}

func RegisteredStop() (bool, error) {
	// Check if we're registered. If not, we don't handle the stop request.
	if registered, err := registered(); err != nil {
		return false, errors.Wrap(err, "unable to determine daemon registration status")
	} else if !registered {
		return false, nil
	}

	// Compute the launchd plist path.
	targetPath := filepath.Join(
		filesystem.HomeDirectory,
		libraryDirectoryName,
		launchAgentsDirectoryName,
		launchdPlistName,
	)

	// Attempt to unload the daemon.
	unload := exec.Command("launchctl", "unload", targetPath)
	if err := unload.Run(); err != nil {
		return false, errors.Wrap(err, "unable to unload launchd plist")
	}

	// Success.
	return true, nil
}
