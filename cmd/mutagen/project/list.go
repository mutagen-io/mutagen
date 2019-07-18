package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd/mutagen/forward"
	"github.com/mutagen-io/mutagen/cmd/mutagen/sync"
	"github.com/mutagen-io/mutagen/pkg/filesystem/locking"
	"github.com/mutagen-io/mutagen/pkg/project"
)

func listMain(command *cobra.Command, arguments []string) error {
	// Compute the name of the configuration file and change our working
	// directory to the path in which the file resides.
	var configurationFileName string
	if len(arguments) == 0 {
		configurationFileName = project.DefaultConfigurationFileName
	} else if len(arguments) == 1 {
		// Parse the target into directory and file name.
		var directory string
		directory, configurationFileName = filepath.Split(arguments[0])
		if configurationFileName == "" {
			return errors.New("empty configuration file name")
		}

		// Switch to the directory (if it's not the current directory).
		if directory != "" {
			if err := os.Chdir(directory); err != nil {
				return errors.Wrap(err, "unable to switch to target directory")
			}
		}
	} else {
		return errors.New("invalid number of arguments")
	}

	// Compute the lock path.
	lockPath := configurationFileName + project.LockFileExtension

	// Create a locker and defer its closure.
	locker, err := locking.NewLocker(lockPath, 0600)
	if err != nil {
		return errors.Wrap(err, "unable to create project locker")
	}
	defer locker.Close()

	// Acquire the project lock and defer its release.
	if err := locker.Lock(true); err != nil {
		return errors.Wrap(err, "unable to acquire project lock")
	}
	defer locker.Unlock()

	// Read the full contents of the lock file. If it's empty, then assume we
	// created it and just remove it.
	buffer := &bytes.Buffer{}
	if length, err := buffer.ReadFrom(locker); err != nil {
		return errors.Wrap(err, "unable to read project lock")
	} else if length == 0 {
		os.Remove(lockPath)
		return errors.New("project not running")
	}
	identifier := buffer.String()

	// Compute the label selector that we're going to use to list sessions.
	labelSelector := fmt.Sprintf("%s=%s", project.LabelKey, identifier)

	// Terminate forwarding sessions.
	if err := forward.ListWithLabelSelector(labelSelector, listConfiguration.long); err != nil {
		return errors.Wrap(err, "unable to list forwarding session(s)")
	}

	// Terminate synchronization sessions.
	if err := sync.ListWithLabelSelector(labelSelector, listConfiguration.long); err != nil {
		return errors.Wrap(err, "unable to list synchronization session(s)")
	}

	// Success.
	return nil
}

var listCommand = &cobra.Command{
	Use:          "list",
	Short:        "Terminate project sessions",
	RunE:         listMain,
	SilenceUsage: true,
}

var listConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
	// long indicates whether or not to use long-format listing.
	long bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := listCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&listConfiguration.help, "help", "h", false, "Show help information")

	// Wire up list flags.
	flags.BoolVarP(&listConfiguration.long, "long", "l", false, "Show detailed session information")
}
