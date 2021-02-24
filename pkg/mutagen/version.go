// +build go1.16

package mutagen

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/pkg/errors"
)

const (
	// VersionMajor represents the current major version of Mutagen.
	VersionMajor = 0
	// VersionMinor represents the current minor version of Mutagen.
	VersionMinor = 12
	// VersionPatch represents the current patch version of Mutagen.
	VersionPatch = 0
	// VersionTag represents a tag to be appended to the Mutagen version string.
	// It must not contain spaces. If empty, no tag is appended to the version
	// string.
	VersionTag = "beta2"
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

// sendVersion writes the current version to the specified writer. Version tag
// components are neither transmitted nor received.
func sendVersion(writer io.Writer) error {
	// Compute the version bytes.
	var data versionBytes
	binary.BigEndian.PutUint32(data[:4], VersionMajor)
	binary.BigEndian.PutUint32(data[4:8], VersionMinor)
	binary.BigEndian.PutUint32(data[8:], VersionPatch)

	// Transmit the bytes.
	_, err := writer.Write(data[:])
	return err
}

// receiveVersion reads version information from the specified reader. Version
// tag components are neither transmitted nor received.
func receiveVersion(reader io.Reader) (uint32, uint32, uint32, error) {
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

// ClientVersionHandshake performs the client side of a version handshake,
// returning an error if the received server version is not compatible with the
// client version.
//
// TODO: Add some ability to support version skew in this function.
func ClientVersionHandshake(connection net.Conn) error {
	// Receive the server's version.
	serverMajor, serverMinor, _, err := receiveVersion(connection)
	if err != nil {
		return errors.Wrap(err, "unable to receive server version")
	}

	// Send our version to the server.
	if err := sendVersion(connection); err != nil {
		return errors.Wrap(err, "unable to send client version")
	}

	// Ensure that our Mutagen versions are compatible. For now, we enforce that
	// they're equal at the minor release level.
	// TODO: Once we lock-in an internal protocol that we're going to support
	// for some time, we can allow some version skew. On the client side in
	// particular, we'll probably want to look out for the specific "locked-in"
	// server protocol that we support and instantiate some frozen client
	// implementation from that version.
	versionMatch := serverMajor == VersionMajor && serverMinor == VersionMinor
	if !versionMatch {
		return errors.New("version mismatch")
	}

	// Success.
	return nil
}

// ServerVersionHandshake performs the server side of a version handshake,
// returning an error if the received client version is not compatible with the
// server version.
//
// TODO: Add some ability to support version skew in this function.
func ServerVersionHandshake(connection net.Conn) error {
	// Send our version to the client.
	if err := sendVersion(connection); err != nil {
		return errors.Wrap(err, "unable to send server version")
	}

	// Receive the client's version.
	clientMajor, clientMinor, _, err := receiveVersion(connection)
	if err != nil {
		return errors.Wrap(err, "unable to receive client version")
	}

	// Ensure that our versions are compatible. For now, we enforce that they're
	// equal at the minor release level.
	versionMatch := clientMajor == VersionMajor && clientMinor == VersionMinor
	if !versionMatch {
		return errors.New("version mismatch")
	}

	// Success.
	return nil
}
