package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	"github.com/havoc-io/mutagen/cmd"
)

var usage = `ci_testing [-h|--help] [-r|--race]
`

func main() {
	// Parse testArguments.
	var race bool
	flagSet := cmd.NewFlagSet("ci_testing", usage, nil)
	flagSet.BoolVarP(&race, "race", "r", false, "enable race detection")
	flagSet.ParseOrDie(os.Args[1:])

	// Create a list of all the packages to test. Go tooling uses "\n" rather
	// than "\r\n", even on Windows, so we don't need to do any replace.
	var packages []string
	list := exec.Command("go", "list", "./...")
	if o, err := list.Output(); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to list test modules"))
	} else {
		packages = strings.Split(strings.TrimSpace(string(o)), "\n")
	}

	// Loop over packages.
	for _, p := range packages {
		// Skip packages that aren't really packages.
		if p == "github.com/havoc-io/mutagen/scripts" {
			continue
		}

		// Set up arguments for testing.
		testArguments := []string{"test", "-v"}
		if race {
			testArguments = append(testArguments, "-race")
		}
		testArguments = append(
			testArguments,
			fmt.Sprintf("-coverprofile=%s.cover", uuid.NewV4().String()),
			p,
		)

		// Execute tests for the package.
		tests := exec.Command("go", testArguments...)
		tests.Stdout = os.Stdout
		tests.Stderr = os.Stderr
		if err := tests.Run(); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to execute tests"))
		}
	}
}
