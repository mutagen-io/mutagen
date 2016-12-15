package session

import (
	"io"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/rpc"
)

const (
	endpointAcceptBacklog = 100

	endpointMethodScan     = "endpoint.Scan"
	endpointMethodStage    = "endpoint.Stage"
	endpointMethodTransmit = "endpoint.Transmit"
	endpointMethodApply    = "endpoint.Apply"
	endpointMethodUpdate   = "endpoint.Update"
)

func ServeEndpoint(stream io.ReadWriteCloser) error {
	// Create a multiplexer.
	multiplexer := multiplex(stream, true)

	// Create an RPC client to connect to the other endpoint.
	client := rpc.NewClient(multiplexer)

	// Create the endpoint.
	endpoint := newEndpoint(client)

	// Create an RPC server.
	server := rpc.NewServer(map[string]rpc.Handler{
		endpointMethodScan:     endpoint.scan,
		endpointMethodStage:    endpoint.stage,
		endpointMethodTransmit: endpoint.transmit,
		endpointMethodApply:    endpoint.apply,
		endpointMethodUpdate:   endpoint.update,
	})

	// Serve RPC requests until there is an error accepting new streams.
	return errors.Wrap(server.Serve(multiplexer), "error serving RPC requests")
}

type endpoint struct {
	client *rpc.Client
	// TODO: Add stager.
}

func newEndpoint(client *rpc.Client) *endpoint {
	return &endpoint{
		client: client,
		// TODO: Add remaining fields.
	}
}

func (e *endpoint) scan(stream *rpc.HandlerStream) {
	// TODO: Implement.
}

func (e *endpoint) stage(stream *rpc.HandlerStream) {
	// TODO: Implement.
}

func (e *endpoint) transmit(stream *rpc.HandlerStream) {
	// TODO: Implement.
}

func (e *endpoint) apply(stream *rpc.HandlerStream) {
	// TODO: Implement.
}

func (e *endpoint) update(stream *rpc.HandlerStream) {
	// TODO: Implement.
}
