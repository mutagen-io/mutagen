package daemon

// The implementation of daemon registration is largely based on these two
// articles:
// https://developer.apple.com/library/content/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html
// https://developer.apple.com/library/content/technotes/tn2083/_index.html#//apple_ref/doc/uid/DTS10003794-CH1-SUBSECTION44

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
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
func Register(logger *logging.Logger) error {
	// If we're already registered, don't do anything.
	if registered, err := registered(); err != nil {
		return fmt.Errorf("unable to determine registration status: %w", err)
	} else if registered {
		return nil
	}

	// Acquire the daemon lock to ensure the daemon isn't running. We switch the
	// start and stop mechanism depending on whether or not we're registered, so
	// we need to make sure we don't try to stop a daemon started using a
	// different mechanism.
	lock, err := AcquireLock(logger)
	if err != nil {
		return errors.New("unable to alter registration while daemon is running")
	}
	defer must.Release(lock, logger)

	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to compute path to home directory: %w", err)
	}

	// Ensure the user's Library directory exists.
	targetPath := filepath.Join(homeDirectory, libraryDirectoryName)
	if err := os.MkdirAll(targetPath, libraryDirectoryPermissions); err != nil {
		return fmt.Errorf("unable to create Library directory: %w", err)
	}

	// Ensure the LaunchAgents directory exists.
	targetPath = filepath.Join(targetPath, launchAgentsDirectoryName)
	if err := os.MkdirAll(targetPath, launchAgentsDirectoryPermissions); err != nil {
		return fmt.Errorf("unable to create LaunchAgents directory: %w", err)
	}

	// Compute the path to the current executable.
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("unable to determine executable path: %w", err)
	}

	// Format a launchd plist.
	plist := fmt.Sprintf(launchdPlistTemplate, executablePath)

	// Attempt to write the launchd plist.
	targetPath = filepath.Join(targetPath, launchdPlistName)
	if err := filesystem.WriteFileAtomic(targetPath, []byte(plist), launchdPlistPermissions, logger); err != nil {
		return fmt.Errorf("unable to write launchd agent plist: %w", err)
	}

	// Success.
	return nil
}

// Unregister performs automatic daemon startup de-registration.
func Unregister(logger *logging.Logger) error {
	// If we're not registered, don't do anything.
	if registered, err := registered(); err != nil {
		return fmt.Errorf("unable to determine registration status: %w", err)
	} else if !registered {
		return nil
	}

	// Acquire the daemon lock to ensure the daemon isn't running. We switch the
	// start and stop mechanism depending on whether or not we're registered, so
	// we need to make sure we don't try to stop a daemon started using a
	// different mechanism.
	lock, err := AcquireLock(logger)
	if err != nil {
		return errors.New("unable to alter registration while daemon is running")
	}
	defer must.Release(lock, logger)

	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to compute path to home directory: %w", err)
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
			return fmt.Errorf("unable to remove launchd agent plist: %w", err)
		}
	}

	// Success.
	return nil
}

// launchctlSpuriousErrorFragment is a fragment of text that appears in spurious
// launchctl load/unload command errors when the daemon run command exits due to
// an existing daemon or the launchctl-hosted daemon isn't running. Ignoring
// this fragment is important for user-friendly idempotency when using launchd
// hosting of the daemon.
const launchctlSpuriousErrorFragment = "failed: 5: Input/output error"

// runLaunchctlIgnoringSpuriousErrors runs a launchctl command and only prints
// out error text if it doesn't contain launchctlSpuriousErrorFragment. The
// standard error stream for the command must not be set.
func runLaunchctlIgnoringSpuriousErrors(command *exec.Cmd) error {
	err := command.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if bytes.Contains(exitErr.Stderr, []byte(launchctlSpuriousErrorFragment)) {
				return nil
			}
		}
	}
	return err
}

// registered determines whether or not automatic daemon startup is currently
// registered.
func registered() (bool, error) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("unable to compute path to home directory: %w", err)
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
		return false, fmt.Errorf("unable to query launchd agent plist: %w", err)
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
		return false, fmt.Errorf("unable to determine daemon registration status: %w", err)
	} else if !registered {
		return false, nil
	}

	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("unable to compute path to home directory: %w", err)
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
	if err := runLaunchctlIgnoringSpuriousErrors(load); err != nil {
		return false, fmt.Errorf("unable to load launchd plist: %w", err)
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
		return false, fmt.Errorf("unable to determine daemon registration status: %w", err)
	} else if !registered {
		return false, nil
	}

	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("unable to compute path to home directory: %w", err)
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
	if err := runLaunchctlIgnoringSpuriousErrors(unload); err != nil {
		return false, fmt.Errorf("unable to unload launchd plist: %w", err)
	}

	// Success.
	return true, nil
}
