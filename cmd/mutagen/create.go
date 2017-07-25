package main

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/session"
	"github.com/havoc-io/mutagen/url"
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
	// Parse command line arguments. The help flag is handled automatically.
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

	// If either URL is a relative path, convert it to an absolute path.
	if alpha.Protocol == url.Protocol_Local {
		if alphaPath, err := filepath.Abs(alpha.Path); err != nil {
			return errors.Wrap(err, "unable to make alpha path absolute")
		} else {
			alpha.Path = alphaPath
		}
	}
	if beta.Protocol == url.Protocol_Local {
		if betaPath, err := filepath.Abs(beta.Path); err != nil {
			return errors.Wrap(err, "unable to make beta path absolute")
		} else {
			beta.Path = betaPath
		}
	}

	// Create a daemon client.
	daemonClient := rpc.NewClient(daemon.NewOpener())

	// Invoke the session creation method and ensure the resulting stream is
	// closed when we're done.
	stream, err := daemonClient.Invoke(session.MethodCreate)
	if err != nil {
		return errors.Wrap(err, "unable to invoke session creation")
	}
	defer stream.Close()

	// Send the initial request.
	if err := stream.Send(session.CreateRequest{
		Alpha:   alpha,
		Beta:    beta,
		Ignores: []string(ignores),
	}); err != nil {
		return errors.Wrap(err, "unable to send creation request")
	}

	// Handle authentication challenges.
	return handleChallengePrompts(stream)
}
