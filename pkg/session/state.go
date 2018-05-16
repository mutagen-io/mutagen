package session

import (
	"github.com/havoc-io/mutagen/pkg/sync"
)

func (s Status) Description() string {
	switch s {
	case Status_Disconnected:
		return "Disconnected"
	case Status_HaltedOnRootDeletion:
		return "Halted due to root deletion"
	case Status_ConnectingAlpha:
		return "Connecting to alpha"
	case Status_ConnectingBeta:
		return "Connecting to beta"
	case Status_Watching:
		return "Watching for changes"
	case Status_ScanningAlpha:
		return "Scanning files on alpha"
	case Status_ScanningBeta:
		return "Scanning files on beta"
	case Status_WaitingForRescan:
		return "Waiting for rescan"
	case Status_Reconciling:
		return "Reconciling changes"
	case Status_StagingAlpha:
		return "Staging files on alpha"
	case Status_StagingBeta:
		return "Staging files on beta"
	case Status_TransitioningAlpha:
		return "Applying changes on alpha"
	case Status_TransitioningBeta:
		return "Applying changes on beta"
	case Status_Saving:
		return "Saving archive"
	default:
		return "Unknown"
	}
}

func (s *State) Copy() *State {
	// Create a shallow copy of the state.
	result := &State{}
	*result = *s

	// Create a shallow copy of the Session member, if present.
	if s.Session != nil {
		result.Session = &Session{}
		*result.Session = *s.Session
	}

	// Create a shallow copy of the StagingStatus member, if present.
	if s.StagingStatus != nil {
		result.StagingStatus = &StagingStatus{}
		*result.StagingStatus = *s.StagingStatus
	}

	// All other composite members are either immutable values or considered to
	// be immutable, so we don't need to copy them.

	// Done.
	return result
}

func convertConflicts(conflicts []sync.Conflict) []*Conflict {
	// If the existing slice is empty, then so is the result.
	if len(conflicts) == 0 {
		return nil
	}

	// Allocate the result.
	result := make([]*Conflict, len(conflicts))

	// Perform conversions.
	for i, c := range conflicts {
		// Create the base conflict.
		conflict := &Conflict{
			AlphaChanges: make([]*Change, len(c.AlphaChanges)),
			BetaChanges:  make([]*Change, len(c.BetaChanges)),
		}
		result[i] = conflict

		// Convert alpha changes.
		for a, alphaChange := range c.AlphaChanges {
			conflict.AlphaChanges[a] = &Change{
				Path: alphaChange.Path,
				Old:  alphaChange.Old.CopyShallow(),
				New:  alphaChange.New.CopyShallow(),
			}
		}

		// Convert beta changes.
		for b, betaChange := range c.BetaChanges {
			conflict.BetaChanges[b] = &Change{
				Path: betaChange.Path,
				Old:  betaChange.Old.CopyShallow(),
				New:  betaChange.New.CopyShallow(),
			}
		}
	}

	// Done.
	return result
}

func convertProblems(problems []sync.Problem) []*Problem {
	// If the existing slice is empty, then so is the result.
	if len(problems) == 0 {
		return nil
	}

	// Allocate the result.
	result := make([]*Problem, len(problems))

	// Perform conversion.
	for i, p := range problems {
		result[i] = &Problem{
			Path:  p.Path,
			Error: p.Error,
		}
	}

	// Done.
	return result
}
