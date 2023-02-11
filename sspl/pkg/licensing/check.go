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
	"context"
	"errors"
	"fmt"
)

// ErrNoLicenseManager is returned by Check when no corresponding license
// manager for the product is available.
var ErrNoLicenseManager = errors.New("no license manager for product")

// Check attempts to find a license manager for the specified product and query
// the license status. It returns whether or not a valid license was found and
// any error that occurred in making that determination. If no license manager
// is registered for the product, then ErrNoLicenseManagerForCheck is returned.
func Check(product string) (bool, error) {
	// Extract the corresponding license manager.
	managerRegistryLock.RLock()
	manager := managerRegistry[product]
	managerRegistryLock.RUnlock()
	if manager == nil {
		return false, ErrNoLicenseManager
	}

	// Perform a poll operation on the manager to determine license status.
	_, state, err := manager.Poll(context.Background(), 0)
	if err != nil {
		return false, fmt.Errorf("unable to poll for license state: %w", err)
	}

	// Success.
	return state.Status == Status_Licensed, nil
}
