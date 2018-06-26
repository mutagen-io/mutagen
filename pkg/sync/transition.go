package sync

import (
	"bytes"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"golang.org/x/text/unicode/norm"

	"github.com/golang/protobuf/ptypes"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// Provider defines the interface that higher-level logic can use to provide
// files to transition algorithms.
type Provider interface {
	// Provide returns a filesystem path to a file containing the contents for
	// the path given as the first argument with the digest specified by the
	// second argument.
	Provide(path string, digest []byte) (string, error)
}

type transitioner struct {
	root             string
	cache            *Cache
	symlinkMode      SymlinkMode
	recomposeUnicode bool
	provider         Provider
	problems         []*Problem
}

func (t *transitioner) recordProblem(path string, err error) {
	t.problems = append(t.problems, &Problem{Path: path, Error: err.Error()})
}

func (t *transitioner) ensureRouteWithProperCase(path string, skipLast bool) error {
	// If the path is empty, then there's nothing to check.
	if path == "" {
		return nil
	}

	// Set the initial parent.
	parent := t.root

	// Decompose the path.
	components := strings.Split(path, "/")

	// Exclude the last component from checking if requested.
	if skipLast && len(components) > 0 {
		components = components[:len(components)-1]
	}

	// While components remain, read the contents of the current parent and
	// ensure that a child with the correct cased entry exists.
	for _, component := range components {
		// Grab the contents for this location.
		contents, err := filesystem.DirectoryContents(parent)
		if err != nil {
			return errors.Wrap(err, "unable to read directory contents")
		}

		// Recompose content names if necessary.
		if t.recomposeUnicode {
			for i, name := range contents {
				contents[i] = norm.NFC.String(name)
			}
		}

		// Check if this path component exists in the contents. It's important
		// to note that the contents are not guaranteed to be ordered, so we
		// can't do a binary search here.
		found := false
		for _, content := range contents {
			if content == component {
				found = true
				break
			}
		}

		// If the component wasn't found, then return an error.
		if !found {
			return errors.New("unable to find matching entry")
		}

		// Update the parent.
		parent = filepath.Join(parent, component)
	}

	// Success.
	return nil
}

func (t *transitioner) ensureExpectedFile(path string, expected *Entry) (os.FileMode, int, int, error) {
	// Grab cache information for this path. If we can't find it, we treat this
	// as an immediate fail. This is a bit of a heuristic/hack, because we could
	// recompute the digest of what's on disk, but for our use case this is very
	// expensive and we SHOULD already have this information cached from the
	// last scan.
	cacheEntry, ok := t.cache.Entries[path]
	if !ok {
		return 0, 0, 0, errors.New("unable to find cache information for path")
	}

	// Grab stat information for this path.
	info, err := os.Lstat(filepath.Join(t.root, path))
	if err != nil {
		return 0, 0, 0, errors.Wrap(err, "unable to grab file statistics")
	}

	// Grab the model.
	mode := info.Mode()

	// Grab the modification time.
	modificationTime := info.ModTime()

	// Grab the cached modification time and convert it to a Go format.
	cachedModificationTime, err := ptypes.Timestamp(cacheEntry.ModificationTime)
	if err != nil {
		return 0, 0, 0, errors.Wrap(err, "unable to convert cached modification time format")
	}

	// If stat information doesn't match, don't bother re-hashing, just abort.
	// Note that we don't really have to check executability here (and we
	// shouldn't since it's not preserved on all systems) - we just need to
	// check that it hasn't changed from the perspective of the filesystem, and
	// that is accomplished as part of the mode check. This is why we don't
	// restrict the mode comparison to the type bits.
	match := os.FileMode(cacheEntry.Mode) == mode &&
		modificationTime.Equal(cachedModificationTime) &&
		cacheEntry.Size == uint64(info.Size()) &&
		bytes.Equal(cacheEntry.Digest, expected.Digest)
	if !match {
		return 0, 0, 0, errors.New("modification detected")
	}

	// Extract ownership.
	uid, gid, err := getOwnership(info)
	if err != nil {
		return 0, 0, 0, errors.Wrap(err, "unable to compute file ownership")
	}

	// Success.
	return mode, uid, gid, nil
}

func (t *transitioner) ensureExpectedSymlink(path string, expected *Entry) error {
	// Grab the link target.
	target, err := os.Readlink(filepath.Join(t.root, path))
	if err != nil {
		return errors.Wrap(err, "unable to read symlink target")
	}

	// If we're in portable symlink mode, then we need to normalize the target
	// coming from disk, because some systems (e.g. Windows) won't round-trip
	// the target correctly.
	if t.symlinkMode == SymlinkMode_SymlinkPortable {
		target, err = normalizeSymlinkAndEnsurePortable(path, target)
		if err != nil {
			return errors.Wrap(err, "unable to normalize target in portable mode")
		}
	}

	// Ensure that the targets match.
	if target != expected.Target {
		return errors.New("symlink target does not match expected")
	}

	// Success.
	return nil
}

func (t *transitioner) ensureNotExists(path string) error {
	// Attempt to grab stat information for the path.
	_, err := os.Lstat(filepath.Join(t.root, path))

	// Handle error cases (which may indicate success).
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "unable to determine path existence")
	}

	// Failure.
	return errors.New("path exists")
}

func (t *transitioner) removeFile(path string, target *Entry) error {
	// Ensure that the existing entry hasn't been modified from what we're
	// expecting.
	if _, _, _, err := t.ensureExpectedFile(path, target); err != nil {
		return errors.Wrap(err, "unable to validate existing file")
	}

	// Remove the file.
	return os.Remove(filepath.Join(t.root, path))
}

func (t *transitioner) removeSymlink(path string, target *Entry) error {
	// Ensure that the existing symlink hasn't been modified from what we're
	// expecting.
	if err := t.ensureExpectedSymlink(path, target); err != nil {
		return errors.Wrap(err, "unable to validate existing symlink")
	}

	// Remove the symlink.
	return os.Remove(filepath.Join(t.root, path))
}

func (t *transitioner) removeDirectory(path string, target *Entry) bool {
	// Compute the full path to this directory.
	fullPath := filepath.Join(t.root, path)

	// List the contents for this directory.
	contentNames, err := filesystem.DirectoryContents(fullPath)
	if err != nil {
		t.recordProblem(path, errors.Wrap(err, "unable to read directory contents"))
		return false
	}

	// Recompose content names if necessary.
	if t.recomposeUnicode {
		for i, name := range contentNames {
			contentNames[i] = norm.NFC.String(name)
		}
	}

	// Loop through contents and remove them. We do this to ensure that what
	// we're removing has the proper case. If we were to just pass the OS what
	// exists in our content map and it were case insensitive, we could delete
	// a file that had been unmodified but renamed. There is, of course, a race
	// condition here between the time we grab the directory contents and the
	// time we remove, but it is very small and we also compare file contents,
	// so the chance of deleting something we shouldn't is very small.
	unknownContentEncountered := false
	for _, name := range contentNames {
		// Compute the content path.
		contentPath := pathpkg.Join(path, name)

		// Grab the corresponding entry. If we don't know anything about this
		// entry, then mark that as a problem and ignore for now.
		entry, ok := target.Contents[name]
		if !ok {
			t.recordProblem(contentPath, errors.New("unknown content encountered on disk"))
			unknownContentEncountered = true
			continue
		}

		// Handle content removal based on type.
		if entry.Kind == EntryKind_Directory {
			if !t.removeDirectory(contentPath, entry) {
				continue
			}
		} else if entry.Kind == EntryKind_File {
			if err = t.removeFile(contentPath, entry); err != nil {
				t.recordProblem(contentPath, errors.Wrap(err, "unable to remove file"))
				continue
			}
		} else if entry.Kind == EntryKind_Symlink {
			if err = t.removeSymlink(contentPath, entry); err != nil {
				t.recordProblem(contentPath, errors.Wrap(err, "unable to remove symlink"))
				continue
			}
		} else {
			t.recordProblem(contentPath, errors.New("unknown entry type found in removal target"))
			continue
		}

		// At this point the removal must have succeeded, so remove the entry
		// from the target.
		delete(target.Contents, name)
	}

	// If we didn't encounter any unknown content and the target contents are
	// empty, then we can attempt to remove the directory itself.
	if !unknownContentEncountered && len(target.Contents) == 0 {
		if err := os.Remove(fullPath); err != nil {
			t.recordProblem(path, errors.Wrap(err, "unable to remove directory"))
		} else {
			return true
		}
	}

	// At this point, we must have encountered some sort of problem earlier, but
	// it will already have been recorded, so we just need to make the removal
	// as failed.
	return false
}

func (t *transitioner) remove(path string, target *Entry) *Entry {
	// If the target is nil, we're done.
	if target == nil {
		return nil
	}

	// Ensure that the path of the target exists (relative to the root) with the
	// specificed casing.
	if err := t.ensureRouteWithProperCase(path, false); err != nil {
		t.recordProblem(path, errors.Wrap(err, "unable to verify path to target"))
		return target
	}

	// Handle removal based on type.
	if target.Kind == EntryKind_Directory {
		// Create a copy of target for mutation.
		targetCopy := target.Copy()

		// Attempt to reduce it.
		if !t.removeDirectory(path, targetCopy) {
			return targetCopy
		}
	} else if target.Kind == EntryKind_File {
		if err := t.removeFile(path, target); err != nil {
			t.recordProblem(path, errors.Wrap(err, "unable to remove file"))
			return target
		}
	} else if target.Kind == EntryKind_Symlink {
		if err := t.removeSymlink(path, target); err != nil {
			t.recordProblem(path, errors.Wrap(err, "unable to remove symlink"))
			return target
		}
	} else {
		t.recordProblem(path, errors.New("removal requested for unknown entry type"))
		return target
	}

	// Success.
	return nil
}

func (t *transitioner) swapFile(path string, oldEntry, newEntry *Entry) error {
	// Compute the full path to this file.
	fullPath := filepath.Join(t.root, path)

	// Ensure that the path of the target exists (relative to the root) with the
	// specificed casing.
	if err := t.ensureRouteWithProperCase(path, false); err != nil {
		return errors.Wrap(err, "unable to verify path to target")
	}

	// Ensure that the existing entry hasn't been modified from what we're
	// expecting.
	mode, uid, gid, err := t.ensureExpectedFile(path, oldEntry)
	if err != nil {
		return errors.Wrap(err, "unable to validate existing file")
	}

	// Compute the new file mode based on the new entry's executability.
	if newEntry.Executable {
		mode = MarkExecutableForReaders(mode)
	} else {
		mode = StripExecutableBits(mode)
	}

	// If both files have the same contents (differing only in permissions),
	// then we won't have staged the file, so we just change the permissions on
	// the existing file.
	if bytes.Equal(oldEntry.Digest, newEntry.Digest) {
		// Attempt to change file permissions.
		if err := os.Chmod(fullPath, mode); err != nil {
			return errors.Wrap(err, "unable to change file permissions")
		}

		// Success.
		return nil
	}

	// Compute the path to the staged file.
	stagedPath, err := t.provider.Provide(path, newEntry.Digest)
	if err != nil {
		return errors.Wrap(err, "unable to locate staged file")
	}

	// Set the mode for the staged file.
	if err := os.Chmod(stagedPath, mode); err != nil {
		return errors.Wrap(err, "unable to set staged file mode")
	}

	// Set the ownership for the staged file.
	if err := setOwnership(stagedPath, uid, gid); err != nil {
		return errors.Wrap(err, "unable to set staged file ownership")
	}

	// Rename the staged file.
	if err := filesystem.RenameFileAtomic(stagedPath, fullPath); err != nil {
		return errors.Wrap(err, "unable to relocate staged file")
	}

	// Success.
	return nil
}

func (t *transitioner) createFile(path string, target *Entry) error {
	// Ensure that the target path doesn't exist, e.g. due to a case conflict or
	// modification since the last scan.
	if err := t.ensureNotExists(path); err != nil {
		return errors.Wrap(err, "unable to ensure path does not exist")
	}

	// Compute the path to the staged file.
	stagedPath, err := t.provider.Provide(path, target.Digest)
	if err != nil {
		return errors.Wrap(err, "unable to locate staged file")
	}

	// Compute the new file mode based on the new entry's executability.
	mode := newFileBaseMode
	if target.Executable {
		mode = MarkExecutableForReaders(mode)
	} else {
		mode = StripExecutableBits(mode)
	}

	// Set the mode for the staged file.
	if err := os.Chmod(stagedPath, mode); err != nil {
		return errors.Wrap(err, "unable to set staged file mode")
	}

	// Rename the staged file.
	if err := filesystem.RenameFileAtomic(stagedPath, filepath.Join(t.root, path)); err != nil {
		return errors.Wrap(err, "unable to relocate staged file")
	}

	// Success.
	return nil
}

func (t *transitioner) createSymlink(path string, target *Entry) error {
	// Ensure that the target path doesn't exist, e.g. due to a case conflict or
	// modification since the last scan.
	if err := t.ensureNotExists(path); err != nil {
		return errors.Wrap(err, "unable to ensure path does not exist")
	}

	// Create the symlink.
	if err := os.Symlink(target.Target, filepath.Join(t.root, path)); err != nil {
		return errors.Wrap(err, "unable to link to target")
	}

	// Success.
	return nil
}

func (t *transitioner) createDirectory(path string, target *Entry) *Entry {
	// Ensure that the target path doesn't exist, e.g. due to a case conflict or
	// modification since the last scan.
	if err := t.ensureNotExists(path); err != nil {
		t.recordProblem(path, errors.Wrap(err, "unable to ensure path does not exist"))
		return nil
	}

	// Attempt to create the directory.
	if err := os.Mkdir(filepath.Join(t.root, path), newDirectoryBaseMode); err != nil {
		t.recordProblem(path, errors.Wrap(err, "unable to create directory"))
		return nil
	}

	// Create a shallow copy of the target that we'll populate as we create its
	// contents.
	created := target.CopyShallow()

	// If there are contents in the target, allocate a map for created, because
	// we'll need to populate it.
	if len(target.Contents) > 0 {
		created.Contents = make(map[string]*Entry)
	}

	// Attempt to create the target contents. Track problems as we go.
	for name, entry := range target.Contents {
		// Compute the content path.
		contentPath := pathpkg.Join(path, name)

		// Handle content creation based on type.
		if entry.Kind == EntryKind_Directory {
			if c := t.createDirectory(contentPath, entry); c != nil {
				created.Contents[name] = c
			}
		} else if entry.Kind == EntryKind_File {
			if err := t.createFile(contentPath, entry); err != nil {
				t.recordProblem(contentPath, errors.Wrap(err, "unable to create file"))
			} else {
				created.Contents[name] = entry
			}
		} else if entry.Kind == EntryKind_Symlink {
			if err := t.createSymlink(contentPath, entry); err != nil {
				t.recordProblem(contentPath, errors.Wrap(err, "unable to create symlink"))
			} else {
				created.Contents[name] = entry
			}
		} else {
			t.recordProblem(contentPath, errors.New("creation requested for unknown entry type"))
		}
	}

	// Return the portion of the target that was created.
	return created
}

func (t *transitioner) create(path string, target *Entry) *Entry {
	// If the target is nil, we're done.
	if target == nil {
		return nil
	}

	// If we're creating something at the root, then ensure that the parent of
	// the root path exists and is a directory. We can assume that it's intended
	// to be a directory since the root is intended to exist inside it.
	if path == "" {
		if err := os.MkdirAll(filepath.Dir(t.root), newDirectoryBaseMode); err != nil {
			t.recordProblem(path, errors.Wrap(err, "unable to create parent component of root path"))
			return nil
		}
	}

	// Ensure that the parent of the target path exists with the proper casing.
	if err := t.ensureRouteWithProperCase(path, true); err != nil {
		t.recordProblem(path, errors.Wrap(err, "unable to verify path to target"))
		return nil
	}

	// Handle creation based on type.
	if target.Kind == EntryKind_Directory {
		return t.createDirectory(path, target)
	} else if target.Kind == EntryKind_File {
		if err := t.createFile(path, target); err != nil {
			t.recordProblem(path, errors.Wrap(err, "unable to create file"))
			return nil
		} else {
			return target
		}
	} else if target.Kind == EntryKind_Symlink {
		if err := t.createSymlink(path, target); err != nil {
			t.recordProblem(path, errors.Wrap(err, "unable to create symlink"))
			return nil
		} else {
			return target
		}
	} else {
		t.recordProblem(path, errors.New("creation requested for unknown entry type"))
		return nil
	}
}

func Transition(
	root string,
	transitions []*Change,
	cache *Cache,
	symlinkMode SymlinkMode,
	recomposeUnicode bool,
	provider Provider,
) ([]*Entry, []*Problem) {
	// Create the transitioner.
	transitioner := &transitioner{
		root:             root,
		cache:            cache,
		symlinkMode:      symlinkMode,
		recomposeUnicode: recomposeUnicode,
		provider:         provider,
	}

	// Set up results.
	var results []*Entry

	// Iterate through transitions.
	for _, t := range transitions {
		// Handle the special case where both old and new are a file. In this
		// case we can do a simple swap. It makes sense to handle this specially
		// because it is a very common case and doing it with a swap will remove
		// any window where the path is empty on the filesystem.
		fileToFile := t.Old != nil && t.New != nil &&
			t.Old.Kind == EntryKind_File &&
			t.New.Kind == EntryKind_File
		if fileToFile {
			if err := transitioner.swapFile(t.Path, t.Old, t.New); err != nil {
				results = append(results, t.Old)
				transitioner.recordProblem(t.Path, errors.Wrap(err, "unable to swap file"))
			} else {
				results = append(results, t.New)
			}
			continue
		}

		// Reduce whatever we expect to see on disk to nil (remove it). If we
		// don't expect to see anything (t.Old == nil), this is a no-op. If this
		// fails, record the reduced entry and continue to the next transition.
		if r := transitioner.remove(t.Path, t.Old); r != nil {
			results = append(results, r)
			continue
		}

		// At this point, we should have nil on disk. Transition to whatever the
		// new entry is (or at least as much of it as we can create). If the new
		// entry is nil, this is a no-op.
		results = append(results, transitioner.create(t.Path, t.New))
	}

	// Done.
	return results, transitioner.problems
}
