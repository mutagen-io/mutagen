package session

import (
	"bytes"
	"crypto/sha1"
)

func checksum(snapshotBytes []byte) []byte {
	result := sha1.Sum(snapshotBytes)
	return result[:]
}

func checksumMatch(snapshotBytes, expected []byte) bool {
	checksum := checksum(snapshotBytes)
	return bytes.Equal(checksum, expected)
}
