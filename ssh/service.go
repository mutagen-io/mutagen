package ssh

import (
	"sync"

	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	"github.com/havoc-io/mutagen/rpc"
)

const (
	MethodPrompt = "ssh.Prompt"
)

type Prompter interface {
	Prompt(string, string) (string, error)
}

type Service struct {
	holdersLock sync.Mutex
	holders     map[string]chan Prompter
}

func NewService() *Service {
	return &Service{
		holders: make(map[string]chan Prompter),
	}
}

func (s *Service) Methods() map[string]rpc.Handler {
	return map[string]rpc.Handler{
		MethodPrompt: s.prompt,
	}
}

func (s *Service) RegisterPrompter(prompter Prompter) string {
	// Generate a unique identifier for this prompter.
	identifier := uuid.NewV4().String()

	// Create and populate a channel for passing the prompter around.
	holder := make(chan Prompter, 1)
	holder <- prompter

	// Register the holder.
	s.holdersLock.Lock()
	s.holders[identifier] = holder
	s.holdersLock.Unlock()

	// Done.
	return identifier
}

func (s *Service) UnregisterPrompter(identifier string) {
	// Grab the holder and deregister it. If it isn't currently registed, this
	// must be a logic error.
	s.holdersLock.Lock()
	holder, ok := s.holders[identifier]
	if !ok {
		panic("deregistration requested for unregistered prompter")
	}
	delete(s.holders, identifier)
	s.holdersLock.Unlock()

	// Get the prompter back and close the holder to let anyone else who has it
	// know that they won't be getting the prompter from it.
	<-holder
	close(holder)
}

type PromptRequest struct {
	Prompter string
	Message  string
	Prompt   string
}

type PromptResponse struct {
	Response string
	Error    string
}

func (s *Service) prompt(stream *rpc.HandlerStream) {
	// Read the request.
	var request PromptRequest
	if stream.Decode(&request) != nil {
		stream.Encode(PromptResponse{Error: "unable to receive request"})
		return
	}

	// Grab the holder for the specified prompter. If this fails, inform the
	// client.
	s.holdersLock.Lock()
	holder, ok := s.holders[request.Prompter]
	s.holdersLock.Unlock()
	if !ok {
		stream.Encode(PromptResponse{Error: "prompter not found"})
		return
	}

	// Acquire the prompter.
	prompter, ok := <-holder
	if !ok {
		stream.Encode(PromptResponse{Error: "unable to acquire prompter"})
		return
	}

	// Prompt.
	response, err := prompter.Prompt(request.Message, request.Prompt)

	// Return the prompter.
	holder <- prompter

	// Handle prompting errors.
	if err != nil {
		stream.Encode(PromptResponse{
			Error: errors.Wrap(err, "unable to prompt").Error(),
		})
		return
	}

	// Success.
	stream.Encode(PromptResponse{Response: response})
}
