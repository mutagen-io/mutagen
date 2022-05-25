package synchronization

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

// Problem represents a synchronization problem.
type Problem struct {
	// Path is the path at which the problem occurred, relative to the
	// synchronization root.
	Path string `json:"path"`
	// Error is a human-readable summary of the problem.
	Error string `json:"error"`
}

// NewProblemFromInternalProblem creates a new problem representation from an
// internal Protocol Buffers representation. The problem must be valid.
func NewProblemFromInternalProblem(problem *core.Problem) *Problem {
	return &Problem{
		Path:  problem.Path,
		Error: problem.Error,
	}
}

// NewProblemSliceFromInternalProblemSlice is a convenience function that calls
// NewProblemFromInternalProblem for a slice of problems.
func NewProblemSliceFromInternalProblemSlice(problems []*core.Problem) []*Problem {
	// If there are no problems, then just return a nil slice.
	count := len(problems)
	if count == 0 {
		return nil
	}

	// Create the resulting slice.
	result := make([]*Problem, count)
	for i := 0; i < count; i++ {
		result[i] = NewProblemFromInternalProblem(problems[i])
	}

	// Done.
	return result
}
