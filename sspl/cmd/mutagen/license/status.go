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
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"

	"github.com/mutagen-io/mutagen/pkg/grpcutil"

	"github.com/mutagen-io/mutagen/sspl/pkg/licensing"
	licensingsvc "github.com/mutagen-io/mutagen/sspl/pkg/service/licensing"
)

// statusMain is the entry point for the status command.
func statusMain(_ *cobra.Command, _ []string) error {
	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to daemon: %w", err)
	}
	defer daemonConnection.Close()

	// Creaate a licensing service client.
	licensingService := licensingsvc.NewLicensingClient(daemonConnection)

	// Perform a status query.
	request := &licensingsvc.StatusRequest{}
	response, err := licensingService.Status(context.Background(), request)
	if err != nil {
		return grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		return fmt.Errorf("invalid status response received: %w", err)
	}

	// Print status information.
	if response.State.Status == licensing.Status_Licensed {
		fmt.Println("Valid Mutagen Pro license")
	} else if response.State.Status == licensing.Status_ValidKey {
		fmt.Println("Valid Mutagen Pro license key, attempting to acquire license")
	} else {
		fmt.Println("No active Mutagen Pro license")
	}

	// Print any warning message from the licensing manager.
	if response.State.Warning != "" {
		cmd.Warning(response.State.Warning)
	}

	// Success.
	return nil
}

// statusCommand is the status command.
var statusCommand = &cobra.Command{
	Use:          "status",
	Short:        "Show Mutagen Pro license status",
	Args:         cmd.DisallowArguments,
	RunE:         statusMain,
	SilenceUsage: true,
}

// statusConfiguration stores configuration for the status command.
var statusConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := statusCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&statusConfiguration.help, "help", "h", false, "Show help information")
}
