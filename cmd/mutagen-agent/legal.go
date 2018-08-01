package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/mutagen"
)

func legalMain(command *cobra.Command, arguments []string) error {
	// Print legal information.
	fmt.Println(mutagen.LegalNotice)

	// Success.
	return nil
}

var legalCommand = &cobra.Command{
	Use:   agent.ModeLegal,
	Short: "Show legal information",
	Run:   cmd.Mainify(legalMain),
}

var legalConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := legalCommand.Flags()
	flags.BoolVarP(&legalConfiguration.help, "help", "h", false, "Show help information")
}
