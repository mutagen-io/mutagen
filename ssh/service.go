package ssh

import (
	"sync"

	"github.com/pkg/errors"

	"golang.org/x/net/context"

	uuid "github.com/satori/go.uuid"
)

type Service struct {
	sync.Mutex
	prompterPassers map[string]chan Prompt_RespondServer
}

func NewService() *Service {
	return &Service{
		prompterPassers: make(map[string]chan Prompt_RespondServer),
	}
}

func (s *Service) Request(_ context.Context, request *PromptRequest) (*PromptResponse, error) {
	// Grab the passer for the specified prompter.
	s.Lock()
	passer, ok := s.prompterPassers[request.Prompter]
	s.Unlock()

	// If there was no passer registered, abort.
	if !ok {
		return nil, errors.New("prompter not found")
	}

	// Wait for the prompter. If we don't receive one, it could be that it
	// disconnected before we could receive it. If we get the prompter, ensure
	// that we return it when we're done.
	prompter, ok := <-passer
	if !ok {
		return nil, errors.New("unable to acquire prompter")
	}
	defer func() {
		passer <- prompter
	}()

	// Forward the request.
	if err := prompter.Send(request); err != nil {
		return nil, errors.Wrap(err, "unable to send prompt")
	}

	// Get the prompt response.
	return prompter.Recv()
}

func (s *Service) Respond(stream Prompt_RespondServer) error {
	// Generate a unique id for this prompter.
	prompter := uuid.NewV4().String()

	// Send the prompter its identifier.
	if err := stream.Send(&PromptRequest{Prompter: prompter}); err != nil {
		return errors.Wrap(err, "unable to send prompter identifier")
	}

	// Grab the context for the stream. The client can use this to signal when
	// it's complete, but it will also be triggered if the client just
	// disconnects, in which case we can abort.
	context := stream.Context()

	// Create the channel that we'll use to pass the prompter around.
	passer := make(chan Prompt_RespondServer, 1)
	passer <- stream

	// Register the passer.
	s.Lock()
	s.prompterPassers[prompter] = passer
	s.Unlock()

	// Wait until the client aborts or disconnects.
	<-context.Done()

	// Get the prompter back.
	<-passer

	// Close the passer to let anyone who currently has it know that they won't
	// be getting the prompter.
	close(passer)

	// Deregister the passer.
	s.Lock()
	delete(s.prompterPassers, prompter)
	s.Unlock()

	// Success.
	return nil
}
