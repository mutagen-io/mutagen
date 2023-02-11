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

	licensingsvc "github.com/mutagen-io/mutagen/sspl/pkg/service/licensing"
)

// deactivateMain is the entry point for the deactivate command.
func deactivateMain(_ *cobra.Command, _ []string) error {
	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to daemon: %w", err)
	}
	defer daemonConnection.Close()

	// Creaate a licensing service client.
	licensingService := licensingsvc.NewLicensingClient(daemonConnection)

	// Perform deactivation.
	request := &licensingsvc.DeactivateRequest{}
	response, err := licensingService.Deactivate(context.Background(), request)
	if err != nil {
		return grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		return fmt.Errorf("invalid deactivate response received: %w", err)
	}

	// Success.
	return nil
}

// deactivateCommand is the deactivate command.
var deactivateCommand = &cobra.Command{
	Use:          "deactivate",
	Short:        "Deactivate any active Mutagen Pro license",
	Args:         cmd.DisallowArguments,
	RunE:         deactivateMain,
	SilenceUsage: true,
}

// deactivateConfiguration stores configuration for the deactivate command.
var deactivateConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := deactivateCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&deactivateConfiguration.help, "help", "h", false, "Show help information")
}
