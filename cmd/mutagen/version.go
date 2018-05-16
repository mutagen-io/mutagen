package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/pkg/mutagen"
)

func versionMain(command *cobra.Command, arguments []string) error {
	// Print version information.
	fmt.Println(mutagen.Version)

	// Success.
	return nil
}

var versionCommand = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run:   mainify(versionMain),
}

var versionConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := versionCommand.Flags()
	flags.BoolVarP(&versionConfiguration.help, "help", "h", false, "Show help information")
}
