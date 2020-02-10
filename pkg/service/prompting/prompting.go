package prompting

import (
	"github.com/pkg/errors"
)

// ensureValid verifies that a PromptRequest is valid.
func (r *PromptRequest) ensureValid() error {
	// A nil prompt request is not valid.
	if r == nil {
		return errors.New("nil prompt request")
	}

	// Verify that the prompter identifier is non-empty.
	if r.Prompter == "" {
		return errors.New("empty prompter identifier")
	}

	// Verify that the prompt is non-empty.
	if r.Prompt == "" {
		return errors.New("empty prompt")
	}

	// Success.
	return nil
}

// EnsureValid verifies that a PromptResponse is valid.
func (r *PromptResponse) EnsureValid() error {
	// A nil prompt response is not valid.
	if r == nil {
		return errors.New("nil prompt response")
	}

	// Any value of the response itself is considered valid.

	// Success.
	return nil
}
