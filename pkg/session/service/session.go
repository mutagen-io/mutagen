package service

import (
	"github.com/pkg/errors"
)

// EnsureValid verifies that a CreateResponse is valid.
func (r *CreateResponse) EnsureValid() error {
	// Ensure that at most a single field is set.
	var fieldsSet int
	if r.Session != "" {
		fieldsSet++
	}
	if r.Message != "" {
		fieldsSet++
	}
	if r.Prompt != "" {
		fieldsSet++
	}
	if fieldsSet > 1 {
		return errors.New("multiple fields set")
	}

	// Success.
	return nil
}

// EnsureValid verifies that a CreateResponse is valid.
func (r *ResumeResponse) EnsureValid() error {
	// Ensure that at most a single field is set.
	var fieldsSet int
	if r.Message != "" {
		fieldsSet++
	}
	if r.Prompt != "" {
		fieldsSet++
	}
	if fieldsSet > 1 {
		return errors.New("multiple fields set")
	}

	// Success.
	return nil
}
