package prompt

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
