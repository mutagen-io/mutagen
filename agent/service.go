package agent

import (
	"net"
	"sync"

	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/url"
)

type Prompter func(*PromptRequest) (*PromptResponse, error)

type Service struct {
	sync.Mutex
	prompters map[string]Prompter
}

func NewService() *Service {
	return &Service{
		prompters: make(map[string]Prompter),
	}
}

func (s *Service) ConnectLocal() (*grpc.ClientConn, error) {
	// Create a gRPC server with the necessary services.
	server := NewServer()

	// Create an in-memory pipe.
	clientConn, serverConn := net.Pipe()

	// Create a one-shot listener and start serving on that listener. This
	// listener will error out after the first accept, but by that time the lone
	// pipe connection will have been accepted and its processing will have
	// started in a separate Goroutine (where the server will live on). This
	// Goroutine will exit when the connection closes.
	listener := NewOneShotListener(serverConn)
	server.Serve(listener)

	// Create a one-shot dialer.
	dialer := &oneShotDialer{clientConn}

	// Attempt to create the gRPC client. If we fail, ensure that the server
	// Goroutine terminates by closing the server connection.
	client, err := grpc.Dial("", grpc.WithBlock(), grpc.WithDialer(dialer.dial), grpc.WithInsecure())
	if err != nil {
		serverConn.Close()
		return nil, errors.Wrap(err, "unable to create ")
	}

	// Success.
	return client, nil
}

func (s *Service) ConnectSSH(url *url.SSHURL, prompter Prompter) (grpc.ClientConn, error) {
	// If a prompter has been provided, create a unique identifier for it and
	// register it. Ensure that it's removed by the time we return from this
	// function.
	var prompterId string
	if prompter != nil {
		prompterId = uuid.NewV4().String()
		s.Lock()
		s.prompters[prompterId] = prompter
		s.Unlock()
		defer func() {
			s.Lock()
			delete(s.prompters, prompterId)
			s.Unlock()
		}()
	}

	// TODO: Implement.
	panic("not implemented")
}

func (s *Service) Prompt(_ context.Context, request *PromptRequest) (*PromptResponse, error) {
	// Look up the specified prompter. If we can't find it, then abort the
	// request, but if we can find it, then remove it from the registry
	// temporarily so that only we can use it, and then defer its re-registry.
	s.Lock()
	prompter, ok := s.prompters[request.Prompter]
	if !ok {
		s.Unlock()
		return nil, errors.New("unable to locate prompter")
	}
	delete(s.prompters, request.Prompter)
	s.Unlock()
	defer func() {
		s.Lock()
		s.prompters[request.Prompter] = prompter
		s.Unlock()
	}()

	// Perform prompting.
	return prompter(request)
}
