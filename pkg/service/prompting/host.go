package prompting

import (
	"context"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/prompting"
)

// Host is a utility function for hosting a prompter via the Prompting service's
// Host method. Although the Host method can be used directly, it requires
// complex interaction and most callers will simply want to host a prompter.
// Prompting is hosted in a background Goroutine. The identifier for the
// prompter is returned, as well as an error channel that will be populated with
// the first error to occur during prompting. Hosting will be terminated when
// either an error occurs or the provided context is cancelled. The error
// channel will be closed after hosting has terminated. If an error occurs
// during hosting setup, then it will be returned and hosting will not commence.
func Host(
	ctx context.Context, client PromptingClient,
	prompter prompting.Prompter, allowPrompts bool,
) (string, <-chan error, error) {
	// Create a subcontext that we can use to perform cancellation in case of a
	// client-side messaging or prompting error.
	ctx, cancel := context.WithCancel(ctx)

	// Initiate hosting.
	stream, err := client.Host(ctx)
	if err != nil {
		cancel()
		return "", nil, fmt.Errorf("unable to initiate prompt hosting: %w", err)
	}

	// Send the initialization request.
	request := &HostRequest{
		AllowPrompts: allowPrompts,
	}
	if err := stream.Send(request); err != nil {
		cancel()
		return "", nil, fmt.Errorf("unable to send initialization request: %w", err)
	}

	// Receive the initialization response, validate it, and extract the
	// prompter identifier.
	var identifier string
	if response, err := stream.Recv(); err != nil {
		cancel()
		return "", nil, fmt.Errorf("unable to receive initialization response: %w", err)
	} else if err = response.EnsureValid(true, allowPrompts); err != nil {
		cancel()
		return "", nil, fmt.Errorf("invalid initialization response received: %w", err)
	} else {
		identifier = response.Identifier
	}

	// Create an error monitoring channel.
	hostingErrors := make(chan error, 1)

	// Start hosting in a background Goroutine.
	go func() {
		// Defer closure of the errors channel.
		defer close(hostingErrors)

		// Defer cancellation of the context to ensure context resource cleanup
		// and server-side cancellation in the event of a client-side error.
		defer cancel()

		// Loop and handle requests indefinitely.
		for {
			if response, err := stream.Recv(); err != nil {
				hostingErrors <- fmt.Errorf("unable to receive message/prompt response: %w",
					grpcutil.PeelAwayRPCErrorLayer(err),
				)
				return
			} else if err = response.EnsureValid(false, allowPrompts); err != nil {
				hostingErrors <- fmt.Errorf("invalid message/prompt response received: %w", err)
				return
			} else if response.IsPrompt {
				if response, err := prompter.Prompt(response.Message); err != nil {
					hostingErrors <- fmt.Errorf("unable to perform prompting: %w", err)
					return
				} else if err = stream.Send(&HostRequest{Response: response}); err != nil {
					hostingErrors <- fmt.Errorf("unable to send prompt response: %w",
						grpcutil.PeelAwayRPCErrorLayer(err),
					)
					return
				}
			} else {
				if err := prompter.Message(response.Message); err != nil {
					hostingErrors <- fmt.Errorf("unable to perform messaging: %w", err)
					return
				} else if err := stream.Send(&HostRequest{}); err != nil {
					hostingErrors <- fmt.Errorf("unable to send message response: %w",
						grpcutil.PeelAwayRPCErrorLayer(err),
					)
					return
				}
			}
		}
	}()

	// Success.
	return identifier, hostingErrors, nil
}
