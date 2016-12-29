package sync

import (
	pathpkg "path"

	"github.com/pkg/errors"
)

type StagingProvider func(string, *Entry) (string, error)

type StagingOperation struct {
	Path  string
	Entry *Entry
}

type stagingOperationFinder struct {
	operations []StagingOperation
}

func (f *stagingOperationFinder) find(path string, entry *Entry) error {
	// If the entry is non-existent, nothing needs to be staged.
	if entry == nil {
		return nil
	}

	// Handle based on type.
	if entry.Kind == EntryKind_File {
		f.operations = append(f.operations, StagingOperation{path, entry})
	} else if entry.Kind == EntryKind_Directory {
		for name, entry := range entry.Contents {
			if err := f.find(pathpkg.Join(path, name), entry); err != nil {
				return err
			}
		}
	} else {
		return errors.New("unknown entry type encountered")
	}

	// Success.
	return nil
}

func StagingOperationsForChanges(changes []Change) ([]StagingOperation, error) {
	// Create an operation finder.
	finder := &stagingOperationFinder{}

	// Have it find operations for all the changes.
	for _, c := range changes {
		if err := finder.find(c.Path, c.New); err != nil {
			return nil, errors.Wrap(err, "unable to find staging operations")
		}
	}

	// Success.
	return finder.operations, nil
}
