package core

import (
	"testing"
)

func TestNilProblemInvalid(t *testing.T) {
	var problem *Problem
	if problem.EnsureValid() == nil {
		t.Error("nil problem treated as valid")
	}
}

func TestProblemValid(t *testing.T) {
	problem := &Problem{Path: "/some/path", Error: "expected error text"}
	if err := problem.EnsureValid(); err != nil {
		t.Error("valid problem failed validation:", err)
	}
}
