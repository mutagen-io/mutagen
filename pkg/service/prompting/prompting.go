package prompting

import (
	"github.com/pkg/errors"
)

// hostRequestMode indicates the mode for a HostRequest.
type hostRequestMode uint8

const (
	// hostRequestModeInitial represents an initial request.
	hostRequestModeInitial hostRequestMode = iota
	// hostRequestModeMessageResponse indicates a response to a message.
	hostRequestModeMessageResponse
	// hostRequestModePromptResponse indicates a response to a prompt.
	hostRequestModePromptResponse
)

// ensureValid verifies that a HostRequest is valid.
func (r *HostRequest) ensureValid(mode hostRequestMode) error {
	// A nil hosting request is not valid.
	if r == nil {
		return errors.New("nil hosting request")
	}

	// Handle validation based on mode.
	if mode == hostRequestModeInitial {
		// Any setting for prompt allowance is valid.

		// Ensure that the response is empty.
		if r.Response != "" {
			return errors.New("unexpected response value on initial request")
		}
	} else {
		// Ensure that prompt allowance hasn't been re-specified.
		if r.AllowPrompts {
			return errors.New("unexpected prompt allowance specification")
		}

		// If responding to a message, ensure that the response is empty. For
		// prompt responses, any value is allowed.
		if mode == hostRequestModeMessageResponse && r.Response != "" {
			return errors.New("unexpected response value when performing messaging")
		}
	}

	// Success.
	return nil
}

// EnsureValid verifies that a HostResponse is valid.
func (r *HostResponse) EnsureValid(first, allowPrompts bool) error {
	// A nil hosting response is not valid.
	if r == nil {
		return errors.New("nil hosting response")
	}

	// Handle validation based on whether or not this is the first response.
	if first {
		// Ensure that the prompter identifier is specified.
		if r.Identifier == "" {
			return errors.New("empty prompter identifier")
		}

		// Ensure that no message type is specified.
		if r.IsPrompt {
			return errors.New("unexpected message type specification")
		}

		// Ensure that no message is provided.
		if r.Message != "" {
			return errors.New("unexpected message")
		}
	} else {
		// Ensure that the prompter identifier isn't specified again.
		if r.Identifier != "" {
			return errors.New("unexpected prompter identifier")
		}

		// Ensure that the message type is allowed.
		if r.IsPrompt && !allowPrompts {
			return errors.New("disallowed prompt message type")
		}

		// Any value of the message is considered valid.
	}

	// Success.
	return nil
}

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

	// Any value of the prompt is considered valid.

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
