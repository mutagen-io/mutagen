//go:build mutagensspl

// Copyright (c) 2022-present Mutagen IO, Inc.
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
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// claims are the claims used by license tokens.
type claims struct {
	// RegisteredClaims are the standard JWT registered claims.
	jwt.RegisteredClaims
	// Product is the product to which the license applies.
	Product string `json:"product"`
}

// Valid implements jwt.Claims.Valid.
func (c *claims) Valid() error {
	// Ensure that required fields are set.
	if c.Issuer == "" {
		return fmt.Errorf("empty or missing license issuer")
	} else if c.Subject == "" {
		return errors.New("empty or missing license subject")
	} else if c.ExpiresAt == nil {
		return errors.New("missing license expiration")
	} else if c.Product == "" {
		return errors.New("empty or missing license product")
	}

	// Call down to the registered claims validation, which will enforce that
	// the license dates are valid.
	return c.RegisteredClaims.Valid()
}

// allowedSigningMethods are the signing methods that we allow in licenses.
var allowedSigningMethods = []string{"EdDSA"}

// parseAndValidateLicenseToken parses and validates a Mutagen-issued license
// token, returning the expiration date for the license and any error that
// occurred during parsing or validation.
func parseAndValidateLicenseToken(token, product string) (time.Time, error) {
	// Create the claims.
	claims := &claims{}

	// Create a parser.
	parser := jwt.NewParser(jwt.WithValidMethods(allowedSigningMethods))

	// Perform parsing and validation.
	if _, err := parser.ParseWithClaims(token, claims, keyLookup); err != nil {
		return time.Time{}, err
	}

	// Verify that the license is for the expected product.
	if claims.Product != product {
		return time.Time{}, errors.New("license token is for incorrect product")
	}

	// Success.
	return claims.ExpiresAt.Time, nil
}
