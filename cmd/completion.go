package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// PerformingShellCompletion indicates whether or not one of Cobra's hidden
// shell completion commands is being used.
var PerformingShellCompletion bool

func init() {
	// Check if one of Cobra's hidden shell completion commands is being used.
	PerformingShellCompletion = len(os.Args) > 1 &&
		(os.Args[1] == cobra.ShellCompRequestCmd ||
			os.Args[1] == cobra.ShellCompNoDescRequestCmd)
}
