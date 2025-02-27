package project

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/cmd/mutagen/forward"
	"github.com/mutagen-io/mutagen/cmd/mutagen/sync"

	"github.com/mutagen-io/mutagen/pkg/filesystem/locking"
	"github.com/mutagen-io/mutagen/pkg/identifier"
	"github.com/mutagen-io/mutagen/pkg/project"
	"github.com/mutagen-io/mutagen/pkg/selection"
)

// terminateMain is the entry point for the terminate command.
func terminateMain(_ *cobra.Command, _ []string) error {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Compute the name of the configuration file and ensure that our working
	// directory is that in which the file resides. This is required for
	// relative paths (including relative synchronization paths and relative
	// Unix Domain Socket paths) to be resolved relative to the project
	// configuration file.
	configurationFileName := project.DefaultConfigurationFileName
	if terminateConfiguration.projectFile != "" {
		var directory string
		directory, configurationFileName = filepath.Split(terminateConfiguration.projectFile)
		if directory != "" {
			if err := os.Chdir(directory); err != nil {
				return fmt.Errorf("unable to switch to target directory: %w", err)
			}
		}
	}

	// Compute the lock path.
	lockPath := configurationFileName + project.LockFileExtension

	// Track whether or not we should remove the lock file on return.
	var removeLockFileOnReturn bool

	// Create a locker and defer its closure and potential removal. On Windows
	// systems, we have to handle this removal after the file is closed.
	locker, err := locking.NewLocker(lockPath, 0600)
	if err != nil {
		return fmt.Errorf("unable to create project locker: %w", err)
	}
	defer func() {
		must.Close(locker, logger)
		if removeLockFileOnReturn && runtime.GOOS == "windows" {
			must.OSRemove(lockPath, logger)
		}
	}()

	// Acquire the project lock and defer its release and potential removal. On
	// Windows systems, we can't remove the lock file if it's locked or even
	// just opened, so we handle removal for Windows systems after we close the
	// lock file (see above). In this case, we truncate the lock file before
	// releasing it to ensure that any other process that opens or acquires the
	// lock file before we manage to remove it will simply see an empty lock
	// file, which it will ignore or attempt to remove.
	if err := locker.Lock(true); err != nil {
		return fmt.Errorf("unable to acquire project lock: %w", err)
	}
	defer func() {
		if removeLockFileOnReturn {
			if runtime.GOOS == "windows" {
				must.Truncate(locker, 0, logger)
			} else {
				must.OSRemove(lockPath, logger)
			}
		}
		must.Unlock(locker, logger)
	}()

	// Read the project identifier from the lock file. If the lock file is
	// empty, then we can assume that we created it when we created the lock and
	// just remove it.
	buffer := &bytes.Buffer{}
	if length, err := buffer.ReadFrom(locker); err != nil {
		return fmt.Errorf("unable to read project lock: %w", err)
	} else if length == 0 {
		removeLockFileOnReturn = true
		return errors.New("project not running")
	}
	projectIdentifier := buffer.String()

	// Ensure that the project identifier is valid.
	if !identifier.IsValid(projectIdentifier) {
		return errors.New("invalid project identifier found in project lock")
	}

	// Load the configuration file.
	configuration, err := project.LoadConfiguration(configurationFileName)
	if err != nil {
		return fmt.Errorf("unable to load configuration file: %w", err)
	}

	// Perform pre-termination commands.
	for _, command := range configuration.BeforeTerminate {
		fmt.Println(">", command)
		if err := runInShell(command); err != nil {
			return fmt.Errorf("pre-terminate command failed: %w", err)
		}
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to daemon: %w", err)
	}
	defer must.Close(daemonConnection, logger)

	// Compute the selection that we're going to use to terminate sessions.
	selection := &selection.Selection{
		LabelSelector: fmt.Sprintf("%s=%s", project.LabelKey, projectIdentifier),
	}

	// Terminate forwarding sessions.
	if err := forward.TerminateWithSelection(daemonConnection, selection); err != nil {
		return fmt.Errorf("unable to terminate forwarding session(s): %w", err)
	}

	// Terminate synchronization sessions.
	if err := sync.TerminateWithSelection(daemonConnection, selection); err != nil {
		return fmt.Errorf("unable to terminate synchronization session(s): %w", err)
	}

	// Perform post-termination commands.
	for _, command := range configuration.AfterTerminate {
		fmt.Println(">", command)
		if err := runInShell(command); err != nil {
			return fmt.Errorf("post-terminate command failed: %w", err)
		}
	}

	// Schedule the project lock for removal.
	removeLockFileOnReturn = true

	// Success.
	return nil
}

// terminateCommand is the terminate command.
var terminateCommand = &cobra.Command{
	Use:          "terminate",
	Short:        "Terminate project sessions",
	Args:         cmd.DisallowArguments,
	RunE:         terminateMain,
	SilenceUsage: true,
}

// terminateConfiguration stores configuration for the terminate command.
var terminateConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// projectFile is the path to the project file, if non-default.
	projectFile string
}

func init() {
	// Grab a handle for the command line flags.
	flags := terminateCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&terminateConfiguration.help, "help", "h", false, "Show help information")

	// Wire up project file flags.
	flags.StringVarP(&terminateConfiguration.projectFile, "project-file", "f", "", "Specify project file")
}
