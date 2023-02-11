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
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"

	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/prompting"

	licensingsvc "github.com/mutagen-io/mutagen/sspl/pkg/service/licensing"
)

// activateMain is the entry point for the activate command.
func activateMain(_ *cobra.Command, _ []string) error {
	// Prompt the user for a license key.
	key, err := prompting.PromptCommandLineWithResponseMode("Enter license key: ", prompting.ResponseModeMasked)
	if err != nil {
		return fmt.Errorf("unable to read license key: %w", err)
	} else if key == "" {
		return errors.New("empty license key")
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to daemon: %w", err)
	}
	defer daemonConnection.Close()

	// Creaate a licensing service client.
	licensingService := licensingsvc.NewLicensingClient(daemonConnection)

	// Perform deactivation.
	request := &licensingsvc.ActivateRequest{Key: key}
	response, err := licensingService.Activate(context.Background(), request)
	if err != nil {
		return grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		return fmt.Errorf("invalid activate response received: %w", err)
	}

	// Success.
	return nil
}

// activateCommand is the activate command.
var activateCommand = &cobra.Command{
	Use:          "activate",
	Short:        "Activate a Mutagen Pro license",
	Args:         cmd.DisallowArguments,
	RunE:         activateMain,
	SilenceUsage: true,
}

// activateConfiguration stores configuration for the activate command.
var activateConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := activateCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&activateConfiguration.help, "help", "h", false, "Show help information")
}
