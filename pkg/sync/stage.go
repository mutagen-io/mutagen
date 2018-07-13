package sync

import (
	"bytes"

	"github.com/pkg/errors"
)

type stagingPathFinder struct {
	paths   []string
	entries []*Entry
}

func (f *stagingPathFinder) find(path string, entry *Entry) error {
	// If the entry is non-existent, nothing needs to be staged.
	if entry == nil {
		return nil
	}

	// Handle based on type.
	if entry.Kind == EntryKind_Directory {
		for name, entry := range entry.Contents {
			if err := f.find(pathJoin(path, name), entry); err != nil {
				return err
			}
		}
	} else if entry.Kind == EntryKind_File {
		f.paths = append(f.paths, path)
		f.entries = append(f.entries, entry)
	} else if entry.Kind == EntryKind_Symlink {
		return nil
	} else {
		return errors.New("unknown entry type encountered")
	}

	// Success.
	return nil
}

// TransitionDependencies analyzes a list of transitions and determines the file
// paths and their corresponding entries that will need to be provided in order
// to apply the transitions using Transition. It guarantees that both returned
// slices will have the same length.
func TransitionDependencies(transitions []*Change) ([]string, []*Entry, error) {
	// Create a path finder.
	finder := &stagingPathFinder{}

	// Have it find paths for all the transitions.
	for _, t := range transitions {
		// If this is a file-to-file transition and only the executability bit
		// is changing, then we don't need to stage, because transition will
		// just modify the target on disk. We only need to watch for these cases
		// when they exist at transition roots (they can't be deeper down in
		// trees).
		fileToFileSameContents := t.Old != nil && t.New != nil &&
			t.Old.Kind == EntryKind_File && t.New.Kind == EntryKind_File &&
			bytes.Equal(t.Old.Digest, t.New.Digest)
		if fileToFileSameContents {
			continue
		}

		// Otherwise we need to perform a full scan.
		if err := finder.find(t.Path, t.New); err != nil {
			return nil, nil, errors.Wrap(err, "unable to find staging paths")
		}
	}

	// Success.
	return finder.paths, finder.entries, nil
}
