package service

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/google/uuid"
)

type Prompter interface {
	Prompt(string, string) (string, error)
}

type Server struct {
	holdersLock sync.Mutex
	holders     map[string]chan Prompter
}

func New() *Server {
	return &Server{
		holders: make(map[string]chan Prompter),
	}
}

func (s *Server) RegisterPrompter(prompter Prompter) (string, error) {
	// Generate a unique identifier for this prompter.
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "unable to generate UUID for prompter")
	}
	identifier := randomUUID.String()

	// Create and populate a channel for passing the prompter around.
	holder := make(chan Prompter, 1)
	holder <- prompter

	// Register the holder.
	s.holdersLock.Lock()
	s.holders[identifier] = holder
	s.holdersLock.Unlock()

	// Done.
	return identifier, nil
}

func (s *Server) UnregisterPrompter(identifier string) {
	// Grab the holder and deregister it. If it isn't currently registered, this
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

type asyncPromptResponse struct {
	response string
	error    error
}

func (s *Server) Prompt(ctx context.Context, request *PromptRequest) (*PromptResponse, error) {
	// Validate the request.
	if request.Prompt == nil {
		return nil, errors.New("invalid prompt request")
	}

	// Grab the holder for the specified prompter. If this fails, inform the
	// client. This won't block for long, if at all, so we don't need to monitor
	// for cancellation here.
	s.holdersLock.Lock()
	holder, ok := s.holders[request.Prompter]
	s.holdersLock.Unlock()
	if !ok {
		return nil, errors.New("prompter not found")
	}

	// Acquire the prompter while watching for cancellation.
	var prompter Prompter
	select {
	case <-ctx.Done():
		return nil, errors.New("prompting cancelled while acquiring prompter")
	case p, ok := <-holder:
		if !ok {
			return nil, errors.New("unable to acquire prompter")
		} else {
			prompter = p
		}
	}

	// Perform prompting while watching for cancellation.
	asyncResponse := make(chan asyncPromptResponse, 1)
	go func() {
		response, err := prompter.Prompt(request.Prompt.Message, request.Prompt.Prompt)
		holder <- prompter
		asyncResponse <- asyncPromptResponse{response, err}
	}()
	select {
	case <-ctx.Done():
		return nil, errors.New("prompting cancelled while waiting for response")
	case r := <-asyncResponse:
		if r.error != nil {
			return nil, errors.Wrap(r.error, "unable to prompt")
		} else {
			return &PromptResponse{Response: r.response}, nil
		}
	}
}
