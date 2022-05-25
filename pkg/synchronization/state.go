package synchronization

import (
	"errors"
	"fmt"
)

// Description returns a human-readable description of the session status.
func (s Status) Description() string {
	switch s {
	case Status_Disconnected:
		return "Waiting to connect"
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

// EnsureValid ensures that State's invariants are respected.
func (s *State) EnsureValid() error {
	// A nil state is not valid.
	if s == nil {
		return errors.New("nil state")
	}

	// We intentionally don't validate the status because we'd have to maintain
	// a pretty large conditional or data structure and we only use it for
	// display anyway, where it'll just render as "Unknown" or similar if it's
	// not valid.

	// Ensure the session is valid.
	if err := s.Session.EnsureValid(); err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}

	// Ensure the staging status is valid.
	if err := s.StagingStatus.EnsureValid(); err != nil {
		return fmt.Errorf("invalid staging status: %w", err)
	}

	// Ensure that all of alpha's scan problem are valid.
	for _, p := range s.AlphaScanProblems {
		if err := p.EnsureValid(); err != nil {
			return fmt.Errorf("invalid alpha scan problem detected: %w", err)
		}
	}

	// Ensure that all of beta's scan problem are valid.
	for _, p := range s.BetaScanProblems {
		if err := p.EnsureValid(); err != nil {
			return fmt.Errorf("invalid beta scan problem detected: %w", err)
		}
	}

	// Ensure that all conflicts are valid.
	for _, c := range s.Conflicts {
		if err := c.EnsureValid(); err != nil {
			return fmt.Errorf("invalid conflict detected: %w", err)
		}
	}

	// Ensure that all of alpha's transition problem are valid.
	for _, p := range s.AlphaTransitionProblems {
		if err := p.EnsureValid(); err != nil {
			return fmt.Errorf("invalid alpha transition problem detected: %w", err)
		}
	}

	// Ensure that all of beta's transition problem are valid.
	for _, p := range s.BetaTransitionProblems {
		if err := p.EnsureValid(); err != nil {
			return fmt.Errorf("invalid beta transition problem detected: %w", err)
		}
	}

	// Ensure that problem and conflict list truncations have only occurred in
	// cases where the corresponding list(s) are non-empty.
	if s.ExcludedAlphaScanProblems > 0 && len(s.AlphaScanProblems) == 0 {
		return errors.New("excluded alpha scan problems reported with no alpha scan problems reported")
	} else if s.ExcludedBetaScanProblems > 0 && len(s.BetaScanProblems) == 0 {
		return errors.New("excluded beta scan problems reported with no beta scan problems reported")
	} else if s.ExcludedConflicts > 0 && len(s.Conflicts) == 0 {
		return errors.New("excluded conflicts reported with no conflicts reported")
	} else if s.ExcludedAlphaTransitionProblems > 0 && len(s.AlphaTransitionProblems) == 0 {
		return errors.New("excluded alpha transition problems reported with no alpha transition problems reported")
	} else if s.ExcludedBetaTransitionProblems > 0 && len(s.BetaTransitionProblems) == 0 {
		return errors.New("excluded beta transition problems reported with no beta transition problems reported")
	}

	// Success.
	return nil
}

// copy creates a shallow copy of the state, deep-copying any mutable members.
func (s *State) copy() *State {
	return &State{
		Session:                         s.Session.copy(),
		Status:                          s.Status,
		AlphaConnected:                  s.AlphaConnected,
		BetaConnected:                   s.BetaConnected,
		LastError:                       s.LastError,
		SuccessfulCycles:                s.SuccessfulCycles,
		StagingStatus:                   s.StagingStatus,
		AlphaScanProblems:               s.AlphaScanProblems,
		ExcludedAlphaScanProblems:       s.ExcludedAlphaScanProblems,
		BetaScanProblems:                s.BetaScanProblems,
		ExcludedBetaScanProblems:        s.ExcludedBetaScanProblems,
		Conflicts:                       s.Conflicts,
		ExcludedConflicts:               s.ExcludedConflicts,
		AlphaTransitionProblems:         s.AlphaTransitionProblems,
		ExcludedAlphaTransitionProblems: s.ExcludedAlphaTransitionProblems,
		BetaTransitionProblems:          s.BetaTransitionProblems,
		ExcludedBetaTransitionProblems:  s.ExcludedBetaTransitionProblems,
	}
}
