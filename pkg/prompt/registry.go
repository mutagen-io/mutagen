package prompt

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/google/uuid"
)

// registryLock is the lock on the global prompter registry.
var registryLock sync.RWMutex

// registry is the global prompter registry.
var registry = make(map[string]chan Prompter)

// RegisterPrompter registers a prompter with the global registry. It generates
// a unique identifier for the prompter that can be used when requesting
// prompting.
func RegisterPrompter(prompter Prompter) (string, error) {
	// Generate a unique identifier for this prompter.
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "unable to generate UUID for prompter")
	}
	identifier := randomUUID.String()

	// Create and populate a channel ("holder") for passing the prompter around.
	holder := make(chan Prompter, 1)
	holder <- prompter

	// Lock the registry for writing and defer its release.
	registryLock.Lock()
	defer registryLock.Unlock()

	// Verify that the identifier isn't currently in use. This is an
	// astronomically small possibility, though it could be indicative of a UUID
	// implementation bug, so we at least watch for it and return an error if it
	// occurs.
	if _, ok := registry[identifier]; ok {
		return "", errors.New("UUID collision")
	}

	// Register the holder.
	registry[identifier] = holder

	// Success.
	return identifier, nil
}

// UnregisterPrompter unregisters a prompter from the global registry. If the
// prompter is not registered, this method panics. If a prompter is unregistered
// with prompts pending for it, they will be cancelled.
func UnregisterPrompter(identifier string) {
	// Lock the registry for writing, grab the holder, and remove it from the
	// registry. If it isn't currently registered, this must be a logic error.
	registryLock.Lock()
	holder, ok := registry[identifier]
	if !ok {
		panic("deregistration requested for unregistered prompter")
	}
	delete(registry, identifier)
	registryLock.Unlock()

	// Get the prompter back and close the holder to let anyone else who has it
	// know that they won't be getting the prompter from it.
	<-holder
	close(holder)
}

// Message invokes the Message method on a prompter in the global registry. If
// the prompter identifier provided is an empty string, this method is a no-op
// and returns a nil error.
func Message(identifier, message string) error {
	// If the prompter identifier is empty, don't do anything.
	if identifier == "" {
		return nil
	}

	// Grab the holder for the specified prompter. We only need a read lock on
	// the registry for this purpose.
	registryLock.RLock()
	holder, ok := registry[identifier]
	registryLock.RUnlock()
	if !ok {
		return errors.New("prompter not found")
	}

	// Acquire the prompter.
	prompter, ok := <-holder
	if !ok {
		return errors.New("unable to acquire prompter")
	}

	// Perform messaging.
	err := prompter.Message(message)

	// Return the prompter to the holder.
	holder <- prompter

	// Handle errors.
	if err != nil {
		errors.Wrap(err, "unable to message")
	}

	// Success.
	return nil
}

// Prompt invokes the Prompt method on a prompter in the global registry.
func Prompt(identifier, prompt string) (string, error) {
	// Grab the holder for the specified prompter. We only need a read lock on
	// the registry for this purpose.
	registryLock.RLock()
	holder, ok := registry[identifier]
	registryLock.RUnlock()
	if !ok {
		return "", errors.New("prompter not found")
	}

	// Acquire the prompter.
	prompter, ok := <-holder
	if !ok {
		return "", errors.New("unable to acquire prompter")
	}

	// Perform prompting.
	response, err := prompter.Prompt(prompt)

	// Return the prompter to the holder.
	holder <- prompter

	// Handle errors.
	if err != nil {
		return "", errors.Wrap(err, "unable to prompt")
	}

	// Success.
	return response, nil
}
