package core

import (
	"errors"
	"sort"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core/fastpath"
)

// EnsureValid ensures that Problem's invariants are respected.
func (p *Problem) EnsureValid() error {
	// A nil problem is not valid.
	if p == nil {
		return errors.New("nil problem")
	}

	// Ensure that an error message has been provided.
	if p.Error == "" {
		return errors.New("empty or missing error message")
	}

	// Success.
	return nil
}

// CopyProblems creates a copy of a list of problems in a new slice, usually for
// the purpose of modifying the list. The problem objects themselves are not
// copied. It preserves nil vs. non-nil characteristics for empty slices.
func CopyProblems(problems []*Problem) []*Problem {
	// If the slice is nil, then preserve its nilness. For zero-length, non-nil
	// slices, we still allocate on the heap to preserve non-nilness.
	if problems == nil {
		return nil
	}

	// Make a copy.
	result := make([]*Problem, len(problems))
	copy(result, problems)

	// Done.
	return result
}

// sortableProblemList implements sort.Interface for problem lists.
type sortableProblemList []*Problem

// Len implements sort.Interface.Len.
func (l sortableProblemList) Len() int {
	return len(l)
}

// Less implements sort.Interface.Less.
func (l sortableProblemList) Less(i, j int) bool {
	return fastpath.Less(l[i].Path, l[j].Path)
}

// Swap implements sort.Interface.Swap.
func (l sortableProblemList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// SortProblems sorts a list of conflicts based on their problem paths.
func SortProblems(conflicts []*Problem) {
	sort.Sort(sortableProblemList(conflicts))
}
