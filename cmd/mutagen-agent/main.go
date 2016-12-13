package main

import (
	"os"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/agent"
	"github.com/havoc-io/mutagen/cmd"
)

var agentUsage = `usage: mutagen-agent should not be manually invoked
`

func main() {
	// Parse flags.
	flagSet := cmd.NewFlagSet("mutagen-agent", agentUsage, []int{1})
	mode := flagSet.ParseOrDie(os.Args[1:])[0]

	// Handle based on mode.
	if mode == "install" {
		if err := agent.Install(); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to install"))
		}
	} else if mode == "endpoint" {
		// TODO: Forward standard input/output to an endpoint server.
	} else {
		cmd.Fatal(errors.Errorf("unknown mode: %s", mode))
	}
}
