package cmd

import (
	"github.com/spf13/cobra"
)

// Mainify is a small utility that wraps a non-standard Cobra entry point (one
// returning an error) and generates a standard Cobra entry point. It's useful
// for entry points to be able to rely on defer-based cleanup, which doesn't
// occur if the entry point terminates the process. This method allows the entry
// point to indicate an error while still performing cleanup.
func Mainify(entry func(*cobra.Command, []string) error) func(*cobra.Command, []string) {
	return func(command *cobra.Command, arguments []string) {
		if err := entry(command, arguments); err != nil {
			Fatal(err)
		}
	}
}
