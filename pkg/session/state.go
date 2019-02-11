package session

import (
	"github.com/pkg/errors"
)

// Description returns a human-readable description of the session status.
func (s Status) Description() string {
	switch s {
	case Status_Disconnected:
		return "Waiting to connect"
	case Status_HaltedOnRootDeletion:
		return "Halted due to root deletion"
	case Status_HaltedOnRootTypeChange:
		return "Halted due to root type change"
	case Status_ConnectingAlpha:
		return "Connecting to alpha"
	case Status_ConnectingBeta:
		return "Connecting to beta"
	case Status_Watching:
		return "Watching for changes"
	case Status_Scanning:
		return "Scanning files"
	case Status_WaitingForRescan:
		return "Waiting 5 seconds for rescan"
	case Status_Reconciling:
		return "Reconciling changes"
	case Status_StagingAlpha:
		return "Staging files on alpha"
	case Status_StagingBeta:
		return "Staging files on beta"
	case Status_Transitioning:
		return "Applying changes"
	case Status_Saving:
		return "Saving archive"
	default:
		return "Unknown"
	}
}

// EnsureValid ensures that State's invariants are respected.
func (s *State) EnsureValid() error {
	// A nil state is not valid.
	if s == nil {
		return errors.New("nil state")
	}

	// We intentionally don't validate the status because we'd have to maintain
	// a pretty large conditional or data structure and we only use it for
	// display anyway, where it'll just render as "Unknown" or similar if it's
	// no valid.

	// Ensure the session is valid.
	if err := s.Session.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid session")
	}

	// Ensure the staging status is valid.
	if err := s.StagingStatus.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid staging status")
	}

	// Ensure that all conflicts are valid.
	for _, c := range s.Conflicts {
		if err := c.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid conflict detected")
		}
	}

	// Ensure that all of alpha's problem are valid.
	for _, c := range s.AlphaProblems {
		if err := c.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid alpha problem detected")
		}
	}

	// Ensure that all of beta's problem are valid.
	for _, c := range s.BetaProblems {
		if err := c.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid beta problem detected")
		}
	}

	// Success.
	return nil
}

// Copy creates a copy of the state, deep-copying those members which are
// mutable.
func (s *State) Copy() *State {
	// Create a shallow copy of the state.
	result := &State{}
	*result = *s

	// Create a shallow copy of the Session member, if present.
	if s.Session != nil {
		result.Session = &Session{}
		*result.Session = *s.Session
	}

	// All other composite members are either immutable values or considered to
	// be immutable, so we don't need to copy them.

	// Done.
	return result
}
