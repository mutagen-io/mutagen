//go:build mutagensspl

package main

import (
	"github.com/mutagen-io/mutagen/sspl/cmd/mutagen/license"
)

// registerLicenseCommand registers the license command tree with the root
// command.
func registerLicenseCommand() {
	rootCommand.AddCommand(license.LicenseCommand)
}
