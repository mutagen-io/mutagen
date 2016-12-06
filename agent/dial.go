package agent

import (
	"github.com/pkg/errors"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/url"
)

type dialResult struct {
	client *grpc.ClientConn
	err    error
}

func Dial(ctx context.Context, remote *url.URL, prompter string) (*grpc.ClientConn, error) {
	// Dial in a separate Goroutine so that we can support context cancellation
	// by simply discarding the result in the event of cancellation.
	results := make(chan dialResult)
	go func() {
		// Handle connection based on the protocol.
		var client *grpc.ClientConn
		var err error
		if remote.Protocol == url.ProtocolLocal {
			client, err = dialLocal()
		} else if remote.Protocol == url.ProtocolSSH {
			client, err = dialSSH(remote, prompter)
		} else {
			err = errors.New("unsupported protocol")
		}

		// Dispatch the result or discard it in the event of cancellation.
		select {
		case results <- dialResult{client, err}:
		case <-ctx.Done():
			if client != nil {
				client.Close()
			}
		}
	}()

	// Wait for dialing to complete or cancellation.
	select {
	case result := <-results:
		return result.client, result.err
	case <-ctx.Done():
		return nil, errors.New("dial cancelled")
	}
}
