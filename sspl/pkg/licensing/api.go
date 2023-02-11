//go:build sspl

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
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

var (
	// errInvalidKey indicates an invalid API key.
	errInvalidKey = errors.New("invalid key")
	// errNoSubscription is returned by getLicenseToken if the user doens't have
	// a valid and healthy subscription.
	errNoSubscription = errors.New("no subscription")
)

// endpoints maps known product identifiers to their corresponding license
// renewal API endpoints.
var endpoints = map[string]string{
	ProductIdentifierMutagenPro: "https://api.mutagen.io/v1/licenses/mutagen-pro",
}

// getLicenseToken attempts to acquire a license token from the Mutagen API
// using the specified endpoint and API token. It handles errors when performing
// the request, but does not parse or validate the license token.
func getLicenseToken(product, apiKey, userAgent string) (string, error) {
	// Look up the product endpoint.
	endpoint, ok := endpoints[product]
	if !ok {
		return "", errors.New("unknown product identifier")
	}

	// Ensure that the API key is viable.
	if apiKey == "" {
		return "", errInvalidKey
	}

	// Compute the user agent, falling back to a default if none is specified.
	if userAgent == "" {
		userAgent = "Mutagen/" + mutagen.Version
	}

	// Create the request.
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create API request: %w", err)
	}
	request.Header.Add("Authorization", "Bearer "+apiKey)
	request.Header.Add("User-Agent", userAgent)

	// Perform the request and defer closure of the response body.
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("unable to perform API request: %w", err)
	}
	defer response.Body.Close()

	// If the response is anything other than OK, then return an error. We'll
	// watch specifically for 401 (Unauthorized) and 402 (Payment Required), but
	// otherwise just return a generic error.
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusUnauthorized {
			return "", errInvalidKey
		} else if response.StatusCode == http.StatusPaymentRequired {
			return "", errNoSubscription
		}
		return "", fmt.Errorf("request returned error code: %d", response.StatusCode)
	}

	// Read the response body in full.
	tokenBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read response body: %w", err)
	} else if !utf8.Valid(tokenBytes) {
		return "", errors.New("received non-UTF-8 response")
	}
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return "", errors.New("empty token")
	}

	// Success.
	return token, nil
}
