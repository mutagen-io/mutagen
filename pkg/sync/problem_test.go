package sync

import (
	"testing"

	"github.com/pkg/errors"
)

func TestNewProblem(t *testing.T) {
	// Create a new problem and ensure that it's non-nil.
	problem := newProblem("/some/path", errors.New("expected error text"))
	if problem == nil {
		t.Fatal("newProblem returned nil problem")
	}

	// Ensure that it matches expected values.
	if problem.Path != "/some/path" {
		t.Error("problem path does not match expected")
	}
	if problem.Error != "expected error text" {
		t.Error("problem error does not match expected")
	}
}

func TestNilProblemInvalid(t *testing.T) {
	var problem *Problem
	if problem.EnsureValid() == nil {
		t.Error("nil problem treated as valid")
	}
}

func TestProblemValid(t *testing.T) {
	problem := newProblem("/some/path", errors.New("expected error text"))
	if err := problem.EnsureValid(); err != nil {
		t.Error("valid problem failed validation:", err)
	}
}
