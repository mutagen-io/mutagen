//go:build sspl && linux

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
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// Version returns the major and minor components of the Linux kernel version.
func Version() (uint64, uint64, error) {
	// Grab system metadata using uname.
	var metadata unix.Utsname
	if err := unix.Uname(&metadata); err != nil {
		return 0, 0, fmt.Errorf("unable to retrieve system metadata: %w", err)
	}

	// Extract the kernel version.
	length := bytes.IndexByte(metadata.Release[:], 0)
	if length == -1 {
		return 0, 0, errors.New("invalid system metadata (missing terminator)")
	}
	version := string(metadata.Release[:length])

	// Parse the kernel version.
	components := strings.SplitN(version, ".", 3)
	if len(components) != 3 {
		return 0, 0, errors.New("unexpected system version format")
	}
	major, err := strconv.ParseUint(components[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to parse major version component: %w", err)
	}
	minor, err := strconv.ParseUint(components[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to parse minor version component: %w", err)
	}

	// Success.
	return major, minor, nil
}
