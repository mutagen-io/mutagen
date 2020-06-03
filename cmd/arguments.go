package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// DisallowArguments is a Cobra arguments validator that disallows positional
// arguments. It is an alternative to cobra.NoArgs, which treats arguments as
// command names and returns a somewhat cryptic error message.
func DisallowArguments(_ *cobra.Command, arguments []string) error {
	if len(arguments) > 0 {
		return errors.New("command does not accept arguments")
	}
	return nil
}
