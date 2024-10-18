package synchronization

import (
	"errors"
	"fmt"
)

// Description returns a human-readable description of the session status.
func (s Status) Description() string {
	switch s {
	case Status_Disconnected:
		return "Disconnected"
	case Status_HaltedOnRootEmptied:
		return "Halted due to one-sided root emptying"
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

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (s Status) MarshalText() ([]byte, error) {
	var result string
	switch s {
	case Status_Disconnected:
		result = "disconnected"
	case Status_HaltedOnRootEmptied:
		result = "halted-on-root-emptied"
	case Status_HaltedOnRootDeletion:
		result = "halted-on-root-deletion"
	case Status_HaltedOnRootTypeChange:
		result = "halted-on-root-type-change"
	case Status_ConnectingAlpha:
		result = "connecting-alpha"
	case Status_ConnectingBeta:
		result = "connecting-beta"
	case Status_Watching:
		result = "watching"
	case Status_Scanning:
		result = "scanning"
	case Status_WaitingForRescan:
		result = "waiting-for-rescan"
	case Status_Reconciling:
		result = "reconciling"
	case Status_StagingAlpha:
		result = "staging-alpha"
	case Status_StagingBeta:
		result = "staging-beta"
	case Status_Transitioning:
		result = "transitioning"
	case Status_Saving:
		result = "saving"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
func (s *Status) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a synchronization status.
	switch text {
	case "disconnected":
		*s = Status_Disconnected
	case "halted-on-root-emptied":
		*s = Status_HaltedOnRootEmptied
	case "halted-on-root-deletion":
		*s = Status_HaltedOnRootDeletion
	case "halted-on-root-type-change":
		*s = Status_HaltedOnRootTypeChange
	case "connecting-alpha":
		*s = Status_ConnectingAlpha
	case "connecting-beta":
		*s = Status_ConnectingBeta
	case "watching":
		*s = Status_Watching
	case "scanning":
		*s = Status_Scanning
	case "waiting-for-rescan":
		*s = Status_WaitingForRescan
	case "reconciling":
		*s = Status_Reconciling
	case "staging-alpha":
		*s = Status_StagingAlpha
	case "staging-beta":
		*s = Status_StagingBeta
	case "transitioning":
		*s = Status_Transitioning
	case "saving":
		*s = Status_Saving
	default:
		return fmt.Errorf("unknown synchronization status: %s", text)
	}

	// Success.
	return nil
}

// ensureValid ensures that EndpointState's invariants are respected.
func (s *EndpointState) ensureValid() error {
	// A nil endpoint state is not valid.
	if s == nil {
		return errors.New("nil state")
	}

	// We could perform additional validation based on the session status and
	// the endpoint connectivity, but it would be prohibitively complex, and all
	// we're really concerned about here is memory safety and other structural
	// invariants.

	// Ensure that all scan problems are valid and truncation is sane.
	for _, p := range s.ScanProblems {
		if err := p.EnsureValid(); err != nil {
			return fmt.Errorf("invalid scan problem detected: %w", err)
		}
	}
	if s.ExcludedScanProblems > 0 && len(s.ScanProblems) == 0 {
		return errors.New("excluded scan problems reported with no scan problems reported")
	}

	// Ensure that all transition problems are valid and truncation is sane.
	for _, p := range s.TransitionProblems {
		if err := p.EnsureValid(); err != nil {
			return fmt.Errorf("invalid transition problem detected: %w", err)
		}
	}
	if s.ExcludedTransitionProblems > 0 && len(s.TransitionProblems) == 0 {
		return errors.New("excluded transition problems reported with no transition problems reported")
	}

	// Ensure that staging progress is valid.
	if err := s.StagingProgress.EnsureValid(); err != nil {
		return fmt.Errorf("invalid staging progress: %w", err)
	}

	// Success.
	return nil
}

// EnsureValid ensures that State's invariants are respected.
func (s *State) EnsureValid() error {
	// A nil state is not valid.
	if s == nil {
		return errors.New("nil state")
	}

	// We could perform additional validation based on the session status, but
	// it would be prohibitively complex, and all we're really concerned about
	// here is memory safety and other structural invariants.

	// Ensure the session is valid.
	if err := s.Session.EnsureValid(); err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}

	// Ensure that all conflicts are valid and truncation is sane.
	for _, c := range s.Conflicts {
		if err := c.EnsureValid(); err != nil {
			return fmt.Errorf("invalid conflict detected: %w", err)
		}
	}
	if s.ExcludedConflicts > 0 && len(s.Conflicts) == 0 {
		return errors.New("excluded conflicts reported with no conflicts reported")
	}

	// Ensure that endpoint states are valid.
	if err := s.AlphaState.ensureValid(); err != nil {
		return fmt.Errorf("invalid alpha endpoint state: %w", err)
	} else if err = s.BetaState.ensureValid(); err != nil {
		return fmt.Errorf("invalid beta endpoint state: %w", err)
	}

	// Success.
	return nil
}
