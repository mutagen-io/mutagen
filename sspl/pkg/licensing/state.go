//go:build mutagensspl

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
)

// EnsureValid ensures that State's invariants are respected.
func (s *State) EnsureValid() error {
	// Ensure that the storage is not nil.
	if s == nil {
		return errors.New("nil state")
	}

	// Ensure that the status is valid.
	if s.Status < Status_Unlicensed || s.Status > Status_Licensed {
		return errors.New("invalid status")
	}

	// Any warning value (including an empty value) is valid.

	// Success.
	return nil
}
