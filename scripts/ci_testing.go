package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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
	listCommand := exec.Command("go", "list", "./...")
	listOutput, err := listCommand.Output()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to list test modules"))
	}
	packages := strings.Split(strings.TrimSpace(string(listOutput)), "\n")

	// Create a temporary directory where we can place coverage profiles and
	// register its cleanup on a best-effort basis (if we die before it's
	// removed, then the OS will deal with it).
	temporaryDirectory, err := ioutil.TempDir("", "coverage")
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create temporary directory"))
	}
	defer os.RemoveAll(temporaryDirectory)

	// Loop over packages and execute tests.
	var coverageProfiles []string
	for _, p := range packages {
		// Skip packages that aren't really packages.
		if p == "github.com/havoc-io/mutagen/scripts" {
			continue
		}

		// Create a unique name for the coverage profile for this package.
		profile := filepath.Join(
			temporaryDirectory,
			fmt.Sprintf("%s.cover", uuid.NewV4().String()),
		)
		coverageProfiles = append(coverageProfiles, profile)

		// Set up arguments for testing.
		testArguments := []string{"test", "-v"}
		if race {
			testArguments = append(testArguments, "-race")
		}
		testArguments = append(
			testArguments,
			fmt.Sprintf("-coverprofile=%s", profile),
			p,
		)

		// Execute tests for the package.
		testCommand := exec.Command("go", testArguments...)
		testCommand.Stdout = os.Stdout
		testCommand.Stderr = os.Stderr
		if err := testCommand.Run(); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to execute tests"))
		}
	}

	// Not all packages have tests, so not all coverage profiles will actually
	// be generated. Filter out those that don't exist.
	var extantCoverageProfiles []string
	for _, p := range coverageProfiles {
		if _, err = os.Stat(p); err != nil {
			if os.IsNotExist(err) {
				continue
			} else {
				cmd.Fatal(errors.Wrap(err, "unable to provide coverage profile"))
			}
		} else {
			extantCoverageProfiles = append(extantCoverageProfiles, p)
		}
	}

	// Invoke gocovmerge to create a combined coverage profile.
	combinedProfile, err := os.Create("coverage.txt")
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create merged coverage profile"))
	}
	defer combinedProfile.Close()
	mergeCommand := exec.Command("gocovmerge", extantCoverageProfiles...)
	mergeCommand.Stdout = combinedProfile
	mergeCommand.Stderr = os.Stderr
	if err := mergeCommand.Run(); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to merge coverge profiles"))
	}
}
