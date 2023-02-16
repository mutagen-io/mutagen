//go:build linux && mutagensspl

// Copyright (c) 2020-present Mutagen IO, Inc.
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

package linux

import (
	"testing"
)

// TestVersion tests that Version succeeds.
func TestVersion(t *testing.T) {
	if major, _, err := Version(); err != nil {
		t.Fatal("unable to query kernel version:", err)
	} else if major == 0 {
		t.Fatal("kernel major version reported as 0")
	}
}
