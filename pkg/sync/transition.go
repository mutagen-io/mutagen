package sync

import (
	"bytes"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// Provider defines the interface that higher-level logic can use to provide
// files to transition algorithms.
type Provider interface {
	// Provide returns a filesystem path to a file containing the contents for
	// the path given as the first argument with the properties specified by the
	// second argument and a mode based on the base mode and entry properties.
	// If a zero base mode is provided, a base mode of ProviderBaseMode should
	// be assumed.
	Provide(path string, entry *Entry, baseMode os.FileMode) (string, error)
}

func ensureRouteWithProperCase(root, path string, skipLast bool) error {
	// If the path is empty, then there's nothing to check.
	if path == "" {
		return nil
	}

	// Set the initial parent.
	parent := root

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

func ensureExpectedFileOrNothing(fullPath, path string, expected *Entry, cache *Cache) (os.FileMode, error) {
	// Grab cache information for this path. If we can't find it, we treat this
	// as an immediate fail. This is a bit of a heuristic/hack, because we could
	// recompute the digest of what's on disk, but for our use case this is very
	// expensive and we SHOULD already have this information cached from the
	// last scan.
	cacheEntry, ok := cache.Entries[path]
	if !ok {
		return 0, errors.New("unable to find cache information for path")
	}

	// Grab stat information for this path.
	info, err := os.Lstat(fullPath)
	if os.IsNotExist(err) {
		return 0, nil
	} else if err != nil {
		return 0, errors.Wrap(err, "unable to grab file statistics")
	}

	// Grab the model.
	mode := info.Mode()

	// Grab the modification time.
	modificationTime := info.ModTime()

	// Grab the cached modification time and convert it to a Go format.
	cachedModificationTime, err := ptypes.Timestamp(cacheEntry.ModificationTime)
	if err != nil {
		return 0, errors.Wrap(err, "unable to convert cached modification time format")
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
		return 0, errors.New("modification detected")
	}

	// Success.
	return mode, nil
}

func ensureExpectedSymlinkOrNothing(root, path string, expected *Entry, symlinkMode SymlinkMode) error {
	// Grab the link target.
	target, err := os.Readlink(filepath.Join(root, path))
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "unable to read symlink target")
	}

	// If we're in sane symlink mode, then we need to normalize the target
	// coming from disk, because some systems (e.g. Windows) won't round-trip
	// the target correctly.
	if symlinkMode == SymlinkMode_Sane {
		target, err = normalizeSymlinkAndEnsureSane(path, target)
		if err != nil {
			return errors.Wrap(err, "unable to normalize target in sane mode")
		}
	}

	// Ensure that the targets match.
	if target != expected.Target {
		return errors.New("symlink target does not match expected")
	}

	// Success.
	return nil
}

func ensureNotExists(fullPath string) error {
	// Attempt to grab stat information for the path.
	_, err := os.Lstat(fullPath)

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

func removeFile(root, path string, target *Entry, cache *Cache) error {
	// Compute the full path to this file.
	fullPath := filepath.Join(root, path)

	// Ensure that the existing entry hasn't been modified from what we're
	// expecting or that it's been removed.
	if _, err := ensureExpectedFileOrNothing(fullPath, path, target, cache); err != nil {
		return errors.Wrap(err, "unable to validate existing file")
	}

	// Remove the file.
	return os.Remove(fullPath)
}

func removeSymlink(root, path string, target *Entry, symlinkMode SymlinkMode) error {
	// Ensure that the existing symlink hasn't been modified from what we're
	// expecting or that it's been removed.
	if err := ensureExpectedSymlinkOrNothing(root, path, target, symlinkMode); err != nil {
		return errors.Wrap(err, "unable to validate existing symlink")
	}

	// Remove the symlink.
	return os.Remove(filepath.Join(root, path))
}

func removeDirectory(root, path string, target *Entry, cache *Cache, symlinkMode SymlinkMode) []*Problem {
	// Compute the full path to this directory.
	fullPath := filepath.Join(root, path)

	// List the contents for this directory.
	contentNames, err := filesystem.DirectoryContents(fullPath)
	if err != nil {
		return []*Problem{newProblem(path, errors.Wrap(err, "unable to read directory contents"))}
	}

	// Loop through contents and remove them. We do this to ensure that what
	// we're removing has the proper case. If we were to just pass the OS what
	// exists in our content map and it were case insensitive, we could delete
	// a file that had been unmodified but renamed. There is, of course, a race
	// condition here between the time we grab the directory contents and the
	// time we remove, but it is very small and we also compare file contents,
	// so the chance of deleting something we shouldn't is very small.
	//
	// Note that we don't need to check that we've removed all entries listed in
	// the target. If they aren't in the directory contents, then they must have
	// already been deleted.
	var problems []*Problem
	for _, name := range contentNames {
		// Compute the content path.
		contentPath := pathpkg.Join(path, name)

		// Grab the corresponding entry. If we don't know anything about this
		// entry, then mark that as a problem and ignore for now.
		entry, ok := target.Contents[name]
		if !ok {
			problems = append(problems, newProblem(
				contentPath,
				errors.New("unknown content encountered on disk"),
			))
			continue
		}

		// Handle its removal accordingly.
		var contentProblems []*Problem
		if entry.Kind == EntryKind_Directory {
			contentProblems = removeDirectory(root, contentPath, entry, cache, symlinkMode)
		} else if entry.Kind == EntryKind_File {
			if err = removeFile(root, contentPath, entry, cache); err != nil {
				contentProblems = append(contentProblems, newProblem(
					contentPath,
					errors.Wrap(err, "unable to remove file"),
				))
			}
		} else if entry.Kind == EntryKind_Symlink {
			if err = removeSymlink(root, contentPath, entry, symlinkMode); err != nil {
				contentProblems = append(contentProblems, newProblem(
					contentPath,
					errors.Wrap(err, "unable to remove symlink"),
				))
			}
		} else {
			contentProblems = append(contentProblems, newProblem(
				contentPath,
				errors.New("unknown entry type found in removal target"),
			))
		}

		// If there weren't any problems, than removal succeeded, so remove this
		// entry from the target. Otherwise add the problems to the complete
		// list.
		if len(contentProblems) == 0 {
			delete(target.Contents, name)
		} else {
			problems = append(problems, contentProblems...)
		}
	}

	// Attempt to remove the directory. If this succeeds, then clear any prior
	// problems, because clearly they no longer matter. This isn't a recursive
	// removal, so if something below failed to delete, this will still fail.
	if err := os.Remove(fullPath); err != nil {
		problems = append(problems, newProblem(
			path,
			errors.Wrap(err, "unable to remove directory"),
		))
	} else {
		problems = nil
	}

	// Done.
	return problems
}

func remove(root, path string, target *Entry, cache *Cache, symlinkMode SymlinkMode) (*Entry, []*Problem) {
	// If the target is nil, we're done.
	if target == nil {
		return nil, nil
	}

	// Ensure that the path of the target exists (relative to the root) with the
	// specificed casing.
	if err := ensureRouteWithProperCase(root, path, false); err != nil {
		return target, []*Problem{newProblem(
			path,
			errors.Wrap(err, "unable to verify path to target"),
		)}
	}

	// Create a copy of target for mutation.
	targetCopy := target.Copy()

	// Check the target type and handle accordingly.
	var problems []*Problem
	if target.Kind == EntryKind_Directory {
		problems = removeDirectory(root, path, targetCopy, cache, symlinkMode)
	} else if target.Kind == EntryKind_File {
		if err := removeFile(root, path, targetCopy, cache); err != nil {
			problems = []*Problem{newProblem(
				path,
				errors.Wrap(err, "unable to remove file"),
			)}
		}
	} else if target.Kind == EntryKind_Symlink {
		if err := removeSymlink(root, path, targetCopy, symlinkMode); err != nil {
			problems = []*Problem{newProblem(
				path,
				errors.Wrap(err, "unable to remove symlink"),
			)}
		}
	} else {
		problems = []*Problem{newProblem(
			path,
			errors.New("removal requested for unknown entry type"),
		)}
	}

	// If there were any problems, then at least the root of the target will
	// have failed to remove, so return the reduced target.
	if len(problems) > 0 {
		return targetCopy, problems
	}

	// Success.
	return nil, nil
}

func swapFile(root, path string, oldEntry, newEntry *Entry, cache *Cache, provider Provider) error {
	// Compute the full path to this file.
	fullPath := filepath.Join(root, path)

	// Ensure that the path of the target exists (relative to the root) with the
	// specificed casing.
	if err := ensureRouteWithProperCase(root, path, false); err != nil {
		return errors.Wrap(err, "unable to verify path to target")
	}

	// Ensure that the existing entry hasn't been modified from what we're
	// expecting.
	baseMode, err := ensureExpectedFileOrNothing(fullPath, path, oldEntry, cache)
	if err != nil {
		return errors.Wrap(err, "unable to validate existing file")
	}

	// Compute the path to the staged file.
	stagedPath, err := provider.Provide(path, newEntry, baseMode)
	if err != nil {
		return errors.Wrap(err, "unable to locate staged file")
	}

	// Rename the staged file.
	if err := filesystem.RenameFileAtomic(stagedPath, fullPath); err != nil {
		return errors.Wrap(err, "unable to relocate staged file")
	}

	// Success.
	return nil
}

func createFile(root, path string, target *Entry, provider Provider) (*Entry, error) {
	// Compute the full path to the target.
	fullPath := filepath.Join(root, path)

	// Ensure that the target path doesn't exist with a case conflict.
	if err := ensureNotExists(fullPath); err != nil {
		return nil, errors.Wrap(err, "case conflict")
	}

	// Compute the path to the staged file.
	stagedPath, err := provider.Provide(path, target, 0)
	if err != nil {
		return nil, errors.Wrap(err, "unable to locate staged file")
	}

	// Rename the staged file.
	if err := filesystem.RenameFileAtomic(stagedPath, fullPath); err != nil {
		return nil, errors.Wrap(err, "unable to relocate staged file")
	}

	// Success.
	return target, nil
}

func createSymlink(root, path string, target *Entry) (*Entry, error) {
	// Compute the full path to the target.
	fullPath := filepath.Join(root, path)

	// Ensure that the target path doesn't exist with a case conflict.
	if err := ensureNotExists(fullPath); err != nil {
		return nil, errors.Wrap(err, "case conflict")
	}

	// Create the symlink.
	if err := os.Symlink(target.Target, fullPath); err != nil {
		return nil, errors.Wrap(err, "unable to link to target")
	}

	// Success.
	return target, nil
}

func createDirectory(root, path string, target *Entry, provider Provider) (*Entry, []*Problem) {
	// Compute the full path to the target.
	fullPath := filepath.Join(root, path)

	// Ensure that the target path doesn't exist with a case conflict.
	if err := ensureNotExists(fullPath); err != nil {
		return nil, []*Problem{newProblem(
			path,
			errors.Wrap(err, "case conflict"),
		)}
	}

	// Attempt to create the directory.
	if err := os.Mkdir(fullPath, directoryBaseMode); err != nil {
		return nil, []*Problem{newProblem(
			path,
			errors.Wrap(err, "unable to create directory"),
		)}
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
	var problems []*Problem
	for name, entry := range target.Contents {
		// Compute the content path.
		contentPath := pathpkg.Join(path, name)

		// Handle content creation based on type.
		var createdContent *Entry
		var contentProblems []*Problem
		if entry.Kind == EntryKind_Directory {
			createdContent, contentProblems = createDirectory(root, contentPath, entry, provider)
		} else if entry.Kind == EntryKind_File {
			var err error
			createdContent, err = createFile(root, contentPath, entry, provider)
			if err != nil {
				contentProblems = append(contentProblems, newProblem(
					contentPath,
					errors.Wrap(err, "unable to create file"),
				))
			}
		} else if entry.Kind == EntryKind_Symlink {
			var err error
			createdContent, err = createSymlink(root, contentPath, entry)
			if err != nil {
				contentProblems = append(contentProblems, newProblem(
					contentPath,
					errors.Wrap(err, "unable to create symlink"),
				))
			}
		} else {
			contentProblems = append(contentProblems, newProblem(
				contentPath,
				errors.New("creation requested for unknown entry type"),
			))
		}

		// If the created content is non-nil, then at least some portion of it
		// was created successfully, so record that.
		if createdContent != nil {
			created.Contents[name] = createdContent
		}

		// Record any problems that occurred when attempting to create the
		// content.
		problems = append(problems, contentProblems...)
	}

	// Return the portion of the target that was created and any problems that
	// occurred.
	return created, problems
}

func create(root, path string, target *Entry, provider Provider) (*Entry, []*Problem) {
	// If the target is nil, we're done.
	if target == nil {
		return nil, nil
	}

	// If we're creating something at the root, then ensure that the parent of
	// the root path exists and is a directory. We can assume that it's intended
	// to be a directory since the root is intented to exist inside it.
	if path == "" {
		if err := os.MkdirAll(filepath.Dir(root), directoryBaseMode); err != nil {
			return nil, []*Problem{newProblem(
				path,
				errors.Wrap(err, "unable to create parent component of root path"),
			)}
		}
	}

	// Ensure that the parent of the target path exists with the proper casing.
	if err := ensureRouteWithProperCase(root, path, true); err != nil {
		return nil, []*Problem{newProblem(
			path,
			errors.Wrap(err, "unable to verify path to target"),
		)}
	}

	// Check the target type and handle accordingly.
	if target.Kind == EntryKind_Directory {
		return createDirectory(root, path, target, provider)
	} else if target.Kind == EntryKind_File {
		if created, err := createFile(root, path, target, provider); err != nil {
			return created, []*Problem{newProblem(
				path,
				errors.Wrap(err, "unable to create file"),
			)}
		} else {
			return created, nil
		}
	} else if target.Kind == EntryKind_Symlink {
		if created, err := createSymlink(root, path, target); err != nil {
			return created, []*Problem{newProblem(
				path,
				errors.Wrap(err, "unable to create symlink"),
			)}
		} else {
			return created, nil
		}
	}
	return nil, []*Problem{newProblem(
		path,
		errors.New("creation requested for unknown entry type"),
	)}
}

func Transition(root string, transitions []*Change, cache *Cache, symlinkMode SymlinkMode, provider Provider) ([]*Entry, []*Problem) {
	// Set up results.
	var results []*Entry
	var problems []*Problem

	// Iterate through transitions.
	for _, t := range transitions {
		// TODO: Should we check for transitions here that don't make any sense
		// but which aren't really logic errors? E.g. it doesn't make sense to
		// have a nil-to-nil transition. Likewise, it doesn't make sense to have
		// a directory-to-directory transition (although it might later on if
		// we're implementing permissions changes). If we see these types of
		// transitions, then there is an error in reconciliation or somewhere
		// else in the synchronization pipeline.

		// Handle the special case where both old and new are a file. In this
		// case we can do a simple swap. It makes sense to handle this specially
		// because it is a very common case and doing it with a swap will remove
		// any window where the path is empty on the filesystem.
		fileToFile := t.Old != nil && t.New != nil &&
			t.Old.Kind == EntryKind_File &&
			t.New.Kind == EntryKind_File
		if fileToFile {
			if err := swapFile(root, t.Path, t.Old, t.New, cache, provider); err != nil {
				results = append(results, t.Old)
				problems = append(problems, newProblem(
					t.Path,
					errors.Wrap(err, "unable to swap file"),
				))
			} else {
				results = append(results, t.New)
			}
			continue
		}

		// Reduce whatever we expect to see on disk to nil (remove it). If we
		// don't expect to see anything (t.Old == nil), this is a no-op. If this
		// fails, record the reduced entry as well as any problems preventing
		// full removal and continue to the next transition.
		if r, p := remove(root, t.Path, t.Old, cache, symlinkMode); r != nil {
			results = append(results, r)
			problems = append(problems, p...)
			continue
		}

		// At this point, we should have nil on disk. Transition to whatever the
		// new entry is. If the new entry is nil, this is a no-op. Record
		// whatever portion of the target we create as well as any problems
		// preventing full creation.
		c, p := create(root, t.Path, t.New, provider)
		results = append(results, c)
		problems = append(problems, p...)
	}

	// Done.
	return results, problems
}
