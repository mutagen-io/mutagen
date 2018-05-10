package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/google/uuid"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/environment"
)

var skippedPackages = []string{
	"github.com/havoc-io/mutagen/cmd/mutagen",
	"github.com/havoc-io/mutagen/cmd/mutagen-agent",
	"github.com/havoc-io/mutagen/scripts",
}

func packageSkipped(pkg string) bool {
	// Check if the package is flagged as skipped.
	for _, p := range skippedPackages {
		if p == pkg {
			return true
		}
	}

	// Otherwise the package isn't skipped.
	return false
}

var usage = `ci_testing [-h|--help] [-r|--race]
`

func main() {
	// Parse command line arguments.
	var race bool
	var goarch string
	var noCover bool
	flagSet := cmd.NewFlagSet("ci_testing", usage, nil)
	flagSet.BoolVar(&race, "race", false, "enable race detection")
	flagSet.StringVar(&goarch, "goarch", "", "override the native GOARCH")
	flagSet.BoolVar(&noCover, "no-cover", false, "disable coverage report")
	flagSet.ParseOrDie(os.Args[1:])

	// Create a list of all the packages to test. Go tooling uses "\n" rather
	// than "\r\n", even on Windows, so we don't need to do any replace.
	listCommand := exec.Command("go", "list", "./...")
	listOutput, err := listCommand.Output()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to list test modules"))
	}
	packages := strings.Split(strings.TrimSpace(string(listOutput)), "\n")

	// If GOARCH is specified, create a copy of the current environment and
	// overwrite GOARCH.
	var testEnvironment []string
	if goarch != "" {
		newEnvironmentMap := environment.CopyCurrent()
		newEnvironmentMap["GOARCH"] = goarch
		testEnvironment = environment.Format(newEnvironmentMap)
	}

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
		// Check if this package is skipped.
		if packageSkipped(p) {
			continue
		}

		// Create a unique name for the coverage profile for this package.
		randomUUID, err := uuid.NewRandom()
		if err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to generate UUID for coverage profile"))
		}
		profile := filepath.Join(
			temporaryDirectory,
			fmt.Sprintf("%s.cover", randomUUID.String()),
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
		testCommand.Env = testEnvironment
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

	// If no coverage report is desired, then we're done.
	if noCover {
		return
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
