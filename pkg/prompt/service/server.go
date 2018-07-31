package prompt

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/google/uuid"
)

// Prompter is the interface to which types supporting prompting must adhere.
type Prompter interface {
	// Message should print a message to the user, returning an error if this is
	// not possible.
	Message(string) error
	// Prompt should print a prompt to the user, returning the user's response
	// or an error if this is not possible.
	Prompt(string) (string, error)
}

// Server provides an implementation of the Prompting service, providing methods
// for prompter registration and internal messaging. This Server is designed to
// operate as a singleton and can be accessed via the global DefaultServer
// variable.
type Server struct {
	// holdersLock locks the holder map.
	holdersLock sync.Mutex
	// holders maps prompter identifiers to a channel that holds the
	// corresponding prompter object.
	holders map[string]chan Prompter
}

// defaultServerLock controls access to the defaultServer variable.
var defaultServerLock sync.RWMutex

// defaultServer is the default prompting server.
var defaultServer *Server

// DefaultServer provides the default prompting server, creating it if
// necessary.
func DefaultServer() *Server {
	// Optimistically attempt to grab the server.
	defaultServerLock.RLock()
	if defaultServer != nil {
		defer defaultServerLock.RUnlock()
		return defaultServer
	}
	defaultServerLock.RUnlock()

	// Otherwise we need to create the server, so we'll need to get a write
	// lock on the server.
	defaultServerLock.Lock()
	defer defaultServerLock.Unlock()

	// It's possible that the server was created by someone else between our two
	// lockings, so see if we can just return it.
	if defaultServer != nil {
		return defaultServer
	}

	// Create the default prompting server.
	defaultServer = &Server{
		holders: make(map[string]chan Prompter),
	}

	// Done.
	return defaultServer
}

// RegisterPrompter registers a prompter with prompting service. It generates a
// unique identifier for the prompter that can be used when requesting prompting
// by the prompting service.
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

// UnregisterPrompter unregisters a prompter registered with the prompting
// service. If the prompter is not registered, this method panics. If a prompter
// is unregistered with prompts pending, they will be cancelled.
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

// Message provides a messaging utility for internal code. If the provided
// prompter identifier is an empty string, then this message is a no-op and will
// return a nil error. This method is not accessible over gRPC, though this may
// change in the future if we delegate custom protocol support to separate
// handlers. In that case, we'll want to follow the same asynchronous,
// cancellable implementation used by Prompt.
func (s *Server) Message(identifier, message string) error {
	// If the prompter identifier is empty, don't do anything.
	if identifier == "" {
		return nil
	}

	// Grab the holder for the specified prompter.
	s.holdersLock.Lock()
	holder, ok := s.holders[identifier]
	s.holdersLock.Unlock()
	if !ok {
		return errors.New("prompter not found")
	}

	// Acquire the prompter while watching for cancellation.
	prompter, ok := <-holder
	if !ok {
		return errors.New("unable to acquire prompter")
	}

	// Perform messaging.
	err := prompter.Message(message)

	// Return the prompter to the holder.
	holder <- prompter

	// Done.
	return errors.Wrap(err, "unable to message")
}

// asyncPromptResponse provides a structure for returning prompt results
// asynchronously, allowing prompting to be cancelled.
type asyncPromptResponse struct {
	// response is the response returned by the prompter.
	response string
	// error is the error returned by the prompter.
	error error
}

// Prompt facilitates prompting by registered prompters.
func (s *Server) Prompt(ctx context.Context, request *PromptRequest) (*PromptResponse, error) {
	// Validate the request.
	if request.Prompter == "" || request.Prompt == "" {
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
		response, err := prompter.Prompt(request.Prompt)
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
