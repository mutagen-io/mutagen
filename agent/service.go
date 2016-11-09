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

	// Create a gRPC client using this connection.
	return clientWithConn(clientConn), nil
}

func (s *Service) ConnectSSH(remote *url.SSHURL, prompter Prompter) (*grpc.ClientConn, error) {
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

	// Attempt a connection. If this fails, but it's a failure that justfies
	// attempting an install, then continue, otherwise fail.
	if conn, install, err := connectSSH(prompterId, remote); err == nil {
		return clientWithConn(conn), nil
	} else if !install {
		return nil, errors.Wrap(err, "unable to connect to agent")
	}

	// Attempt to install.
	if err := installSSH(prompterId, remote); err != nil {
		return nil, errors.Wrap(err, "unable to install agent")
	}

	// Re-attempt connectivity.
	if conn, _, err := connectSSH(prompterId, remote); err != nil {
		return nil, errors.Wrap(err, "unable to connect to agent")
	} else {
		return clientWithConn(conn), nil
	}
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
