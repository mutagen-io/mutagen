package daemon

// The implementation of daemon registration is largely based on these two
// articles:
// https://developer.apple.com/library/content/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html
// https://developer.apple.com/library/content/technotes/tn2083/_index.html#//apple_ref/doc/uid/DTS10003794-CH1-SUBSECTION44

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// RegistrationSupported indicates whether or not daemon registration is
// supported on this platform.
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
	// libraryDirectoryName is the name of the Library directory inside the
	// user's home directory.
	libraryDirectoryName = "Library"
	// libraryDirectoryPermissions are the permissions to use for Library
	// directory creation in the event that it does not exist.
	libraryDirectoryPermissions = 0700

	// launchAgentsDirectoryName is the name of the LaunchAgents directory
	// inside the Library directory.
	launchAgentsDirectoryName = "LaunchAgents"
	// launchAgentsDirectoryPermissions are the permissions to use for
	// LaunchAgents directory creation in the event that it does not exist.
	launchAgentsDirectoryPermissions = 0755

	// launchdPlistName is the name of the launchd property list file to create
	// to register the daemon for automatic startup.
	launchdPlistName = "io.mutagen.mutagen.plist"
	// launchdPlistPermissions are the permissions to use for the launchd
	// property list file.
	launchdPlistPermissions = 0644
)

// Register performs automatic daemon startup registration.
func Register() error {
	// If we're already registered, don't do anything.
	if registered, err := registered(); err != nil {
		return errors.Wrap(err, "unable to determine registration status")
	} else if registered {
		return nil
	}

	// Acquire the daemon lock to ensure the daemon isn't running. We switch the
	// start and stop mechanism depending on whether or not we're registered, so
	// we need to make sure we don't try to stop a daemon started using a
	// different mechanism.
	lock, err := AcquireLock()
	if err != nil {
		return errors.New("unable to alter registration while daemon is running")
	}
	defer lock.Release()

	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "unable to compute path to home directory")
	}

	// Ensure the user's Library directory exists.
	targetPath := filepath.Join(homeDirectory, libraryDirectoryName)
	if err := os.MkdirAll(targetPath, libraryDirectoryPermissions); err != nil {
		return errors.Wrap(err, "unable to create Library directory")
	}

	// Ensure the LaunchAgents directory exists.
	targetPath = filepath.Join(targetPath, launchAgentsDirectoryName)
	if err := os.MkdirAll(targetPath, launchAgentsDirectoryPermissions); err != nil {
		return errors.Wrap(err, "unable to create LaunchAgents directory")
	}

	// Compute the path to the current executable.
	executablePath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "unable to determine executable path")
	}

	// Format a launchd plist.
	plist := fmt.Sprintf(launchdPlistTemplate, executablePath)

	// Attempt to write the launchd plist.
	targetPath = filepath.Join(targetPath, launchdPlistName)
	if err := filesystem.WriteFileAtomic(targetPath, []byte(plist), launchdPlistPermissions); err != nil {
		return errors.Wrap(err, "unable to write launchd agent plist")
	}

	// Success.
	return nil
}

// Unregister performs automatic daemon startup de-registration.
func Unregister() error {
	// If we're not registered, don't do anything.
	if registered, err := registered(); err != nil {
		return errors.Wrap(err, "unable to determine registration status")
	} else if !registered {
		return nil
	}

	// Acquire the daemon lock to ensure the daemon isn't running. We switch the
	// start and stop mechanism depending on whether or not we're registered, so
	// we need to make sure we don't try to stop a daemon started using a
	// different mechanism.
	lock, err := AcquireLock()
	if err != nil {
		return errors.New("unable to alter registration while daemon is running")
	}
	defer lock.Release()

	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "unable to compute path to home directory")
	}

	// Compute the launchd plist path.
	targetPath := filepath.Join(
		homeDirectory,
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

// registered determines whether or not automatic daemon startup is currently
// registered.
func registered() (bool, error) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return false, errors.Wrap(err, "unable to compute path to home directory")
	}

	// Compute the launchd plist path.
	targetPath := filepath.Join(
		homeDirectory,
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

// RegisteredStart potentially handles daemon start operations if the daemon is
// registered for automatic start with the system. It returns false if the start
// operation was not handled and should be handled by the normal start command.
func RegisteredStart() (bool, error) {
	// Check if we're registered. If not, we don't handle the start request.
	if registered, err := registered(); err != nil {
		return false, errors.Wrap(err, "unable to determine daemon registration status")
	} else if !registered {
		return false, nil
	}

	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return false, errors.Wrap(err, "unable to compute path to home directory")
	}

	// Compute the launchd plist path.
	targetPath := filepath.Join(
		homeDirectory,
		libraryDirectoryName,
		launchAgentsDirectoryName,
		launchdPlistName,
	)

	// Attempt to load the daemon.
	load := exec.Command("launchctl", "load", targetPath)
	load.Stdout = os.Stdout
	load.Stderr = os.Stderr
	if err := load.Run(); err != nil {
		return false, errors.Wrap(err, "unable to load launchd plist")
	}

	// Success.
	return true, nil
}

// RegisteredStop potentially handles stop start operations if the daemon is
// registered for automatic start with the system. It returns false if the stop
// operation was not handled and should be handled by the normal stop command.
func RegisteredStop() (bool, error) {
	// Check if we're registered. If not, we don't handle the stop request.
	if registered, err := registered(); err != nil {
		return false, errors.Wrap(err, "unable to determine daemon registration status")
	} else if !registered {
		return false, nil
	}

	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return false, errors.Wrap(err, "unable to compute path to home directory")
	}

	// Compute the launchd plist path.
	targetPath := filepath.Join(
		homeDirectory,
		libraryDirectoryName,
		launchAgentsDirectoryName,
		launchdPlistName,
	)

	// Attempt to unload the daemon.
	unload := exec.Command("launchctl", "unload", targetPath)
	unload.Stdout = os.Stdout
	unload.Stderr = os.Stderr
	if err := unload.Run(); err != nil {
		return false, errors.Wrap(err, "unable to unload launchd plist")
	}

	// Success.
	return true, nil
}
