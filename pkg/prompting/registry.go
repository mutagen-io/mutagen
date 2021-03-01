package prompting

import (
	"errors"
	"fmt"
	"sync"

	"github.com/mutagen-io/mutagen/pkg/identifier"
)

// registryLock is the lock on the global prompter registry.
var registryLock sync.RWMutex

// registry is the global prompter registry.
var registry = make(map[string]chan Prompter)

// RegisterPrompter registers a prompter with the global registry. It
// automatically generates a unique identifier for the prompter.
func RegisterPrompter(prompter Prompter) (string, error) {
	// Generate a unique identifier for this prompter.
	identifier, err := identifier.New(identifier.PrefixPrompter)
	if err != nil {
		return "", fmt.Errorf("unable to generate prompter identifier: %w", err)
	}

	// Perform registration.
	if err := RegisterPrompterWithIdentifier(identifier, prompter); err != nil {
		return "", err
	}

	// Success.
	return identifier, nil
}

// RegisterPrompterWithIdentifier registers a prompter with the global registry
// using the specified identifier.
func RegisterPrompterWithIdentifier(identifier string, prompter Prompter) error {
	// Enforce that the identifier is non-empty.
	if identifier == "" {
		return errors.New("empty identifier")
	}

	// Create and populate a "holder" (channel) for passing the prompter around.
	holder := make(chan Prompter, 1)
	holder <- prompter

	// Lock the registry for writing and defer its release.
	registryLock.Lock()
	defer registryLock.Unlock()

	// Check for identifier collisions. This won't be a problem with our
	// internally generated identifiers, but since this method accepts arbitrary
	// identifiers, we want to be sure to avoid collisions.
	if _, ok := registry[identifier]; ok {
		return errors.New("identifier collision")
	}

	// Register the holder.
	registry[identifier] = holder

	// Success.
	return nil
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
		return fmt.Errorf("unable to message: %w", err)
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
		return "", fmt.Errorf("unable to prompt: %w", err)
	}

	// Success.
	return response, nil
}
