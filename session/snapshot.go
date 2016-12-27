package session

import (
	"bytes"
	"crypto/sha1"
	"io"
	"sort"

	"github.com/golang/protobuf/proto"

	"github.com/havoc-io/mutagen/rsync"
	"github.com/havoc-io/mutagen/sync"
)

// byName provides the sort interface for StableEntryContent, sorting by name.
type byName []*StableEntryContent

func (n byName) Len() int {
	return len(n)
}

func (n byName) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n byName) Less(i, j int) bool {
	return n[i].Name < n[j].Name
}

func stableCopy(entry *sync.Entry) *StableEntry {
	// If the entry is nil, then the copy is nil.
	if entry == nil {
		return nil
	}

	// Create the result.
	result := &StableEntry{
		Kind:       entry.Kind,
		Executable: entry.Executable,
		Digest:     entry.Digest,
	}

	// Copy contents.
	for name, entry := range entry.Contents {
		result.Contents = append(result.Contents, &StableEntryContent{
			Name:  name,
			Entry: stableCopy(entry),
		})
	}

	// Sort contents by name.
	sort.Sort(byName(result.Contents))

	// Done.
	return result
}

func stableMarshal(entry *sync.Entry) ([]byte, error) {
	// Convert the entry to a stable copy.
	stableEntry := stableCopy(entry)

	// Wrap it in an archive in case it's nil.
	stableArchive := &StableArchive{Root: stableEntry}

	// Attempt to marshal.
	return proto.Marshal(stableArchive)
}

func stableUnmarshal(encoded []byte) (*sync.Entry, error) {
	// We can unmarshal directly into a normal archive since they are
	// byte-compatible with stable archives.
	archive := &Archive{}
	if err := proto.Unmarshal(encoded, archive); err != nil {
		return nil, err
	}

	// Success.
	return archive.Root, nil
}

func snapshotChecksum(snapshotBytes []byte) []byte {
	result := sha1.Sum(snapshotBytes)
	return result[:]
}

func snapshotChecksumMatch(snapshotBytes, expectedSnapshotChecksum []byte) bool {
	return bytes.Equal(snapshotChecksum(snapshotBytes), expectedSnapshotChecksum)
}

func snapshotSignature(baseSnapshotBytes []byte) ([]rsync.BlockHash, error) {
	// Create an rsyncer.
	rsyncer := rsync.New()

	// Wrap up the base snapshot bytes in a reader.
	base := bytes.NewReader(baseSnapshotBytes)

	// Compute the signature.
	return rsyncer.Signature(base)
}

func deltafySnapshot(
	snapshotBytes []byte,
	baseSnapshotSignature []rsync.BlockHash,
) ([]rsync.Operation, error) {
	// Create an empty result.
	var delta []rsync.Operation

	// Create an operation transmitter to populate the result. Note that we copy
	// any operation data buffers because the rsync package re-uses them.
	transmit := func(operation rsync.Operation) error {
		// Copy the operation's data buffer if necessary.
		if len(operation.Data) > 0 {
			dataCopy := make([]byte, len(operation.Data))
			copy(dataCopy, operation.Data)
			operation.Data = dataCopy
		}

		// Record the operation.
		delta = append(delta, operation)

		// Success.
		return nil
	}

	// Create an rsyncer.
	rsyncer := rsync.New()

	// Wrap up the snapshot bytes in a reader.
	reader := bytes.NewReader(snapshotBytes)

	// Compute the delta.
	if err := rsyncer.Deltafy(reader, baseSnapshotSignature, transmit); err != nil {
		return nil, err
	}

	// Success.
	return delta, nil
}

func patchSnapshot(baseSnapshotBytes []byte, delta []rsync.Operation) ([]byte, error) {
	// Create an rsyncer.
	rsyncer := rsync.New()

	// Wrap up the base snapshot bytes in a reader.
	base := bytes.NewReader(baseSnapshotBytes)

	// Create an output buffer.
	output := bytes.NewBuffer(nil)

	// Create an operation receiver that will return delta operations.
	receive := func() (rsync.Operation, error) {
		// If there are operations remaining, return the next one and reduce.
		if len(delta) > 0 {
			result := delta[0]
			delta = delta[1:]
			return result, nil
		}

		// Otherwise we're done.
		return rsync.Operation{}, io.EOF
	}

	// Perform application.
	if err := rsyncer.Patch(output, base, receive, nil); err != nil {
		return nil, err
	}

	// Success.
	return output.Bytes(), nil
}
