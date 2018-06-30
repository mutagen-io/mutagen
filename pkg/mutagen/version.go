// +build go1.10

package mutagen

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// VersionMajor represents the current major version of Mutagen.
	VersionMajor = 0
	// VersionMinor represents the current minor version of Mutagen.
	VersionMinor = 4
	// VersionPatch represents the current patch version of Mutagen.
	VersionPatch = 1
)

var Version string

func init() {
	Version = fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}

type versionBytes [12]byte

func SendVersion(writer io.Writer) error {
	// Compute the version bytes.
	var data versionBytes
	binary.BigEndian.PutUint32(data[:4], VersionMajor)
	binary.BigEndian.PutUint32(data[4:8], VersionMinor)
	binary.BigEndian.PutUint32(data[8:], VersionPatch)

	// Transmit the bytes.
	_, err := writer.Write(data[:])
	return err
}

func ReceiveVersion(reader io.Reader) (uint32, uint32, uint32, error) {
	// Read the bytes.
	var data versionBytes
	if _, err := io.ReadFull(reader, data[:]); err != nil {
		return 0, 0, 0, err
	}

	// Decode components.
	major := binary.BigEndian.Uint32(data[:4])
	minor := binary.BigEndian.Uint32(data[4:8])
	patch := binary.BigEndian.Uint32(data[8:])

	// Done.
	return major, minor, patch, nil
}

func ReceiveAndCompareVersion(reader io.Reader) (bool, error) {
	// Receive the version.
	major, minor, patch, err := ReceiveVersion(reader)
	if err != nil {
		return false, err
	}

	// Compare the version.
	return major == VersionMajor &&
		minor == VersionMinor &&
		patch == VersionPatch, nil
}
