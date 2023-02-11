//go:build sspl

// Copyright (c) 2023-present Mutagen IO, Inc.
//
// This program is free software: you can redistribute it and/or modify it under
// the terms of the Server Side Public License, version 1, as published by
// MongoDB, Inc.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE. See the Server Side Public License for more
// details.
//
// You should have received a copy of the Server Side Public License along with
// this program. If not, see
// <http://www.mongodb.com/licensing/server-side-public-license>.

package license

import (
	"github.com/spf13/cobra"
)

// licenseMain is the entry point for the license command.
func licenseMain(command *cobra.Command, _ []string) error {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()

	// Success.
	return nil
}

// LicenseCommand is the license command.
var LicenseCommand = &cobra.Command{
	Use:          "license",
	Short:        "Manage Mutagen Pro licensing",
	RunE:         licenseMain,
	SilenceUsage: true,
}

// licenseConfiguration stores configuration for the license command.
var licenseConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := LicenseCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&licenseConfiguration.help, "help", "h", false, "Show help information")

	// Register commands.
	LicenseCommand.AddCommand(
		activateCommand,
		deactivateCommand,
		statusCommand,
	)
}
