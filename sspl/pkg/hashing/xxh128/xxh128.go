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

package xxh128

import (
	"hash"

	"github.com/zeebo/xxh3"
)

// xxh128Hash implements hash.Hash using the XXH128 algorithm.
type xxh128Hash struct {
	// Hasher is the underlying hasher.
	*xxh3.Hasher
}

// New returns a new XXH128 hash.
func New() hash.Hash {
	return &xxh128Hash{xxh3.New()}
}

// Sum implements hash.Hash.Sum.
func (h *xxh128Hash) Sum(b []byte) []byte {
	// Compute the sum and associated bytes.
	sum128 := h.Sum128()
	sum128Bytes := sum128.Bytes()

	// If b is nil, then take the fast way out.
	if b == nil {
		return sum128Bytes[:]
	}

	// Otherwise append the bytes to b.
	return append(b, sum128Bytes[:]...)
}
