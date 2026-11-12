package prompting

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"github.com/mutagen-io/mutagen/pkg/prompting"
)

// testHostStream is a Prompting_HostServer that records whether the prompter
// identifier is already registered at the time it is sent to the client.
type testHostStream struct {
	// ServerStream satisfies the remaining stream methods, which Host does not
	// use.
	grpc.ServerStream
	// ctx is the stream context.
	ctx context.Context
	// cancel terminates the stream context.
	cancel context.CancelFunc
	// identifier is the identifier sent in the initial response.
	identifier string
	// registeredAtSend indicates whether the identifier was registered when
	// the initial response was sent.
	registeredAtSend bool
}

// Recv implements Prompting_HostServer.Recv.
func (s *testHostStream) Recv() (*HostRequest, error) {
	return &HostRequest{}, nil
}

// Send implements Prompting_HostServer.Send.
func (s *testHostStream) Send(response *HostResponse) error {
	// Record the identifier.
	s.identifier = response.Identifier

	// Attempt to register a placeholder prompter with the same identifier. A
	// collision means that Host registered its prompter before sending the
	// identifier.
	if err := prompting.RegisterPrompterWithIdentifier(s.identifier, nil); err != nil {
		s.registeredAtSend = true
	} else {
		prompting.UnregisterPrompter(s.identifier)
	}

	// Terminate the stream.
	s.cancel()

	// Success.
	return nil
}

// Context implements Prompting_HostServer.Context.
func (s *testHostStream) Context() context.Context {
	return s.ctx
}

// TestHostRegistersBeforeSendingIdentifier tests that Host registers its
// prompter before the client can receive (and use) the prompter identifier.
func TestHostRegistersBeforeSendingIdentifier(t *testing.T) {
	// Create the stream.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := &testHostStream{ctx: ctx, cancel: cancel}

	// Perform hosting.
	if err := NewServer().Host(stream); err != nil {
		t.Fatal("hosting failed:", err)
	}

	// Verify that the identifier was registered when it was sent.
	if stream.identifier == "" {
		t.Fatal("no identifier sent")
	} else if !stream.registeredAtSend {
		t.Error("prompter not registered when identifier was sent")
	}

	// Verify that the prompter was unregistered.
	if err := prompting.Message(stream.identifier, "test"); err == nil {
		t.Error("prompter still registered after hosting completed")
	}
}
