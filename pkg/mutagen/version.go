// +build go1.12

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
	VersionMinor = 9
	// VersionPatch represents the current patch version of Mutagen.
	VersionPatch = 0
	// VersionTag represents a tag to be appended to the Mutagen version string.
	// It must not contain spaces. If empty, no tag is appended to the version
	// string.
	VersionTag = "dev"
)

// Version provides a stringified version of the current Mutagen version.
var Version string

// init performs global initialization.
func init() {
	// Compute the stringified version.
	if VersionTag != "" {
		Version = fmt.Sprintf("%d.%d.%d-%s", VersionMajor, VersionMinor, VersionPatch, VersionTag)
	} else {
		Version = fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
	}
}

// versionBytes is a type that can be used to send and receive version
// information over the wire.
type versionBytes [12]byte

// SendVersion writes the current Mutagen version to the specified writer.
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

// ReceiveVersion reads version information from the specified reader.
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

// ReceiveAndCompareVersion reads version information from the specified reader
// and ensures that it matches the current Mutagen version.
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
