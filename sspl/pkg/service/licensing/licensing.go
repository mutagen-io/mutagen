//go:build sspl

// Copyright (c) 2023-present Mutagen IO, Inc.
//
// This program is free software: you can redistribute it and/or modify it under
// the terms of the Server Side Public License, version 1, as published by
// MongoDB, Inc.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE. See the Server Side Public License for more
// details.
//
// You should have received a copy of the Server Side Public License along with
// this program. If not, see
// <http://www.mongodb.com/licensing/server-side-public-license>.

package licensing

import (
	"errors"
	"fmt"
)

// ensureValid verifies that an ActivateRequest is valid.
func (r *ActivateRequest) ensureValid() error {
	// Ensure that the request is not nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Ensure that a token has been provided. We don't validate its format,
	// which will instead be done by the API. All we can ensure is that it's
	// non-empty, but it is otherwise opaque.
	if r.Key == "" {
		return errors.New("empty key")
	}

	// Success.
	return nil
}

// EnsureValid verifies that an ActivateResponse is valid.
func (r *ActivateResponse) EnsureValid() error {
	// Ensure that the response is not nil.
	if r == nil {
		return errors.New("nil response")
	}

	// Success.
	return nil
}

// ensureValid verifies that a StatusRequest is valid.
func (r *StatusRequest) ensureValid() error {
	// Ensure that the request is not nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Success.
	return nil
}

// EnsureValid verifies that a StatusResponse is valid.
func (r *StatusResponse) EnsureValid() error {
	// Ensure that the response is not nil.
	if r == nil {
		return errors.New("nil response")
	}

	// Ensure that the state is valid.
	if err := r.State.EnsureValid(); err != nil {
		return fmt.Errorf("invalid licensing state: %w", err)
	}

	// Success.
	return nil
}

// ensureValid verifies that a DeactivateRequest is valid.
func (r *DeactivateRequest) ensureValid() error {
	// Ensure that the request is not nil.
	if r == nil {
		return errors.New("nil request")
	}

	// Success.
	return nil
}

// EnsureValid verifies that a DeactivateResponse is valid.
func (r *DeactivateResponse) EnsureValid() error {
	// Ensure that the response is not nil.
	if r == nil {
		return errors.New("nil response")
	}

	// Success.
	return nil
}
