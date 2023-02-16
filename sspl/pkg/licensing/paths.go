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

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

const (
	// keyStorageName is the name of the API key storage file within the
	// product's license data storage directory.
	keyStorageName = "key"
	// licenseStorageName is the name of the license token storage file within
	// the product's license data storage directory.
	licenseStorageName = "license"
)

// pathToProductStorage computes the path to a product's license storage
// directory and ensures that the directory exists.
func pathToProductStorage(product string) (string, error) {
	// Ensure that the product specification is valid.
	//
	// TODO: We should probably perform more aggressive validation here,
	// probably with some sort of reverse-DNS notation parsing, but this is fine
	// for now since this an internal API.
	if product == "" {
		return "", errors.New("invalid product identifier")
	}

	// Compute the path and ensure that it exists.
	return filesystem.Mutagen(true, filesystem.MutagenLicensingDirectoryName, product)
}
