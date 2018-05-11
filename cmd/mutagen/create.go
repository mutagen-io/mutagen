package main

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/rpc"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/url"
)

var createUsage = `usage: mutagen create [-h|--help] [-i|--ignore=<pattern>]
                      <alpha> <beta>

Creates and starts a new synchronization session.
`

type ignorePatterns []string

func (p *ignorePatterns) String() string {
	return "ignore patterns"
}

func (p *ignorePatterns) Set(value string) error {
	*p = append(*p, value)
	return nil
}

func createMain(arguments []string) error {
	// Parse command line arguments.
	var ignores ignorePatterns
	flagSet := cmd.NewFlagSet("create", createUsage, []int{2})
	flagSet.VarP(&ignores, "ignore", "i", "specify ignore paths")
	urls := flagSet.ParseOrDie(arguments)

	// Extract and parse URLs.
	alpha, err := url.Parse(urls[0])
	if err != nil {
		return errors.Wrap(err, "unable to parse alpha URL")
	}
	beta, err := url.Parse(urls[1])
	if err != nil {
		return errors.Wrap(err, "unable to parse beta URL")
	}

	// If either URL is a local path, make sure it's normalized.
	if alpha.Protocol == url.Protocol_Local {
		if alphaPath, err := filesystem.Normalize(alpha.Path); err != nil {
			return errors.Wrap(err, "unable to normalize alpha path")
		} else {
			alpha.Path = alphaPath
		}
	}
	if beta.Protocol == url.Protocol_Local {
		if betaPath, err := filesystem.Normalize(beta.Path); err != nil {
			return errors.Wrap(err, "unable to normalize beta path")
		} else {
			beta.Path = betaPath
		}
	}

	// Create a daemon client.
	daemonClient := rpc.NewClient(daemon.NewOpener())

	// Invoke the session creation method and ensure the resulting stream is
	// closed when we're done.
	stream, err := daemonClient.Invoke(sessionpkg.MethodCreate)
	if err != nil {
		return errors.Wrap(err, "unable to invoke session creation")
	}
	defer stream.Close()

	// Send the initial request.
	request := sessionpkg.CreateRequest{
		Alpha:   alpha,
		Beta:    beta,
		Ignores: []string(ignores),
	}
	if err := stream.Send(request); err != nil {
		return errors.Wrap(err, "unable to send creation request")
	}

	// Handle authentication challenges.
	if err := handlePromptRequests(stream); err != nil {
		return errors.Wrap(err, "unable to handle prompt requests")
	}

	// Receive the create response.
	var response sessionpkg.CreateResponse
	if err := stream.Receive(&response); err != nil {
		return errors.Wrap(err, "unable to receive create response")
	}

	// Print the session identifier.
	fmt.Println("Created session", response.Session)

	// Success.
	return nil
}
