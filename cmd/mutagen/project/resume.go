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

func resumeMain(command *cobra.Command, arguments []string) error {
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

	// Compute the label selector that we're going to use to resume sessions.
	labelSelector := fmt.Sprintf("%s=%s", project.LabelKey, identifier)

	// Resume forwarding sessions.
	if err := forward.ResumeWithLabelSelector(labelSelector); err != nil {
		return errors.Wrap(err, "unable to resume forwarding session(s)")
	}

	// Resume synchronization sessions.
	if err := sync.ResumeWithLabelSelector(labelSelector); err != nil {
		return errors.Wrap(err, "unable to resume synchronization session(s)")
	}

	// Success.
	return nil
}

var resumeCommand = &cobra.Command{
	Use:          "resume [<configuration-file>]",
	Short:        "Resume project sessions",
	RunE:         resumeMain,
	SilenceUsage: true,
}

var resumeConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := resumeCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&resumeConfiguration.help, "help", "h", false, "Show help information")
}
