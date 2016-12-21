package sync

import (
	"bytes"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes"

	"github.com/havoc-io/mutagen/filesystem"
)

func ensureRouteWithProperCase(root, path string, skipLast bool) error {
	// If the path is empty, then there's nothing to check.
	if path == "" {
		return nil
	}

	// Set the initial parent.
	parent := root

	// Decompose the path.
	components := strings.Split(path, "/")

	// If we were requested to not check the last component, then remove it from
	// the list.
	if skipLast && len(components) > 0 {
		components = components[:len(components)-1]
	}

	// While components remain, read the contents of the current parent and
	// ensure that a child with the correct cased entry exists.
	for len(components) > 0 {
		// Grab the contents for this location.
		contents, err := filesystem.DirectoryContents(parent)
		if err != nil {
			return errors.Wrap(err, "unable to read directory contents")
		}

		// Ensure that this path exists in contents. We can do a binary search
		// since contents will be sorted. If there's not a match, we're done.
		index := sort.SearchStrings(contents, components[0])
		if index == len(contents) || contents[index] != components[0] {
			return errors.New("unable to find matching entry")
		}

		// Update the parent.
		parent = filepath.Join(root, components[0])

		// Reduce the component list.
		components = components[1:]
	}

	// Success.
	return nil
}

func ensureExpected(fullPath, path string, target *Entry, cache *Cache) error {
	// Grab cache information for this path. If we can't find it, we treat this
	// as an immediate fail. This is a bit of a heuristic/hack, because we could
	// recompute the digest of what's on disk, but for our use case this is very
	// expensive and we SHOULD already have this information cached from the
	// last scan.
	cacheEntry, ok := cache.Entries[path]
	if !ok {
		return errors.New("unable to find cache information for path")
	}

	// Grab stat information for this path.
	info, err := os.Lstat(fullPath)
	if err != nil {
		return errors.Wrap(err, "unable to grab file statistics")
	}

	// Convert the modification time to Protocol Buffers format.
	modificationTime, err := ptypes.TimestampProto(info.ModTime())
	if err != nil {
		return errors.Wrap(err, "unable to convert modification timestamp")
	}

	// If stat information doesn't match, don't bother re-hashing, just abort.
	// Note that we don't really have to check executability here (and we
	// shouldn't since it's not preserved on all systems) - we just need to
	// check that it hasn't changed from the perspective of the filesystem, and
	// that is accomplished as part of the mode check. This is why we don't
	// restrict the mode comparison to the type bits.
	match := os.FileMode(cacheEntry.Mode) == info.Mode() &&
		timestampsEqual(cacheEntry.ModificationTime, modificationTime) &&
		cacheEntry.Size == uint64(info.Size()) &&
		bytes.Equal(cacheEntry.Digest, target.Digest)
	if !match {
		return errors.New("modification detected")
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
	// expecting.
	if err := ensureExpected(fullPath, path, target, cache); err != nil {
		return errors.Wrap(err, "unable to validate existing file")
	}

	// Remove the file.
	return os.Remove(fullPath)
}

func removeDirectory(root, path string, target *Entry, cache *Cache) error {
	// Compute the full path to this directory.
	fullPath := filepath.Join(root, path)

	// List the contents for this directory.
	contentNames, err := filesystem.DirectoryContents(fullPath)
	if err != nil {
		return errors.Wrap(err, "unable to read directory contents")
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
	for _, c := range contentNames {
		// Grab the corresponding entry.
		entry, ok := target.Find(c)
		if !ok {
			return errors.Wrap(err, "unknown directory content encountered")
		}

		// Compute its path.
		entryPath := pathpkg.Join(path, c)

		// Handle its removal accordingly.
		if entry.Kind == EntryKind_Directory {
			err = removeDirectory(root, entryPath, entry, cache)
		} else if entry.Kind == EntryKind_File {
			err = removeFile(root, entryPath, entry, cache)
		} else {
			err = errors.New("unknown entry type found in removal target")
		}

		// If there was an error, abort.
		if err != nil {
			return err
		}

		// Otherwise, remove this entry from the target. This must succeed.
		if !target.Remove(c) {
			panic("failed to remove path that was previously found")
		}
	}

	// Remove the directory.
	return os.Remove(fullPath)
}

func remove(root, path string, target *Entry, cache *Cache) (*Entry, error) {
	// If the target is nil, we're done.
	if target == nil {
		return nil, nil
	}

	// Ensure that the path of the target exists (relative to the root) with the
	// specificed casing.
	if err := ensureRouteWithProperCase(root, path, false); err != nil {
		return target, errors.Wrap(err, "unable to verify path to target")
	}

	// Create a copy of target for mutation.
	targetCopy := target.copy()

	// Check the target type and handle accordingly.
	var err error
	if target.Kind == EntryKind_Directory {
		err = removeDirectory(root, path, targetCopy, cache)
	} else if target.Kind == EntryKind_File {
		err = removeFile(root, path, targetCopy, cache)
	} else {
		err = errors.New("removal requested for unknown entry type")
	}

	// If there was an error, we need to return the target copy that has been
	// mutated into what remains.
	if err != nil {
		return targetCopy, err
	}

	// Success.
	return nil, nil
}

func swap(root, path string, oldEntry, newEntry *Entry, cache *Cache, provider StagingProvider) error {
	// Compute the full path to this file.
	fullPath := filepath.Join(root, path)

	// Ensure that the path of the target exists (relative to the root) with the
	// specificed casing.
	if err := ensureRouteWithProperCase(root, path, false); err != nil {
		return errors.Wrap(err, "unable to verify path to target")
	}

	// Ensure that the existing entry hasn't been modified from what we're
	// expecting.
	if err := ensureExpected(fullPath, path, oldEntry, cache); err != nil {
		return errors.Wrap(err, "unable to validate existing file")
	}

	// Compute the path to the staged file.
	stagedPath, err := provider.Provide(path, newEntry)
	if err != nil {
		return errors.Wrap(err, "unable to locate staged file")
	}

	// Rename the staged file.
	if err := os.Rename(stagedPath, fullPath); err != nil {
		return errors.Wrap(err, "unable to relocate staged file")
	}

	// Success.
	return nil
}

func createFile(root, path string, target *Entry, provider StagingProvider) (*Entry, error) {
	// Compute the full path to the target.
	fullPath := filepath.Join(root, path)

	// Compute the path to the staged file.
	stagedPath, err := provider.Provide(path, target)
	if err != nil {
		return nil, errors.Wrap(err, "unable to locate staged file")
	}

	// Rename the staged file.
	if err := os.Rename(stagedPath, fullPath); err != nil {
		return nil, errors.Wrap(err, "unable to relocate staged file")
	}

	// Success.
	return target, nil
}

func createDirectory(root, path string, target *Entry, provider StagingProvider) (*Entry, error) {
	// Compute the full path to the target.
	fullPath := filepath.Join(root, path)

	// Attempt to create the directory.
	if err := os.Mkdir(fullPath, 0700); err != nil {
		return nil, errors.Wrap(err, "unable to create directory")
	}

	// Create a shallow copy of the target that we'll populate as we create its
	// contents.
	created := target.copyShallow()

	// Attempt to create the target contents.
	var err error
	for _, content := range target.Contents {
		// Handle content creation based on type.
		var createdContent *Entry
		if content.Entry.Kind == EntryKind_Directory {
			createdContent, err = createDirectory(
				root, pathpkg.Join(path, content.Name), content.Entry, provider,
			)
		} else if content.Entry.Kind == EntryKind_File {
			createdContent, err = createFile(
				root, pathpkg.Join(path, content.Name), content.Entry, provider,
			)
		} else {
			err = errors.New("unknown entry type found in creation target")
		}

		// If the created content is non-nil, then at least some portion of it
		// was created successfully, so record that.
		if createdContent != nil {
			created.Insert(content.Name, createdContent)
		}

		// If there was an error, abort.
		if err != nil {
			break
		}
	}

	// Return the created target and any error that occurred.
	return created, err
}

func create(root, path string, target *Entry, provider StagingProvider) (*Entry, error) {
	// If the target is nil, we're done.
	if target == nil {
		return nil, nil
	}

	// Ensure that the parent of the target path exists with the proper casing.
	if err := ensureRouteWithProperCase(root, path, true); err != nil {
		return nil, errors.Wrap(err, "unable to verify path to target")
	}

	// Compute the full path to this file.
	fullPath := filepath.Join(root, path)

	// Ensure that the target path doesn't exist.
	if err := ensureNotExists(fullPath); err != nil {
		return nil, errors.Wrap(err, "unable to ensure path does not exist")
	}

	// Check the target type and handle accordingly.
	if target.Kind == EntryKind_Directory {
		return createDirectory(root, path, target, provider)
	} else if target.Kind == EntryKind_File {
		return createFile(root, path, target, provider)
	}
	return nil, errors.New("creation requested for unknown entry type")
}

func Transition(root string, transitions []Change, cache *Cache, provider StagingProvider) ([]Change, []Problem) {
	// Set up results.
	var results []Change
	var problems []Problem

	// Iterate through transitions.
	for _, t := range transitions {
		// TODO: Should we check for logic errors here that don't prevent
		// execution? E.g. if doesn't make sense to have a nil-to-nil
		// transition. Likewise, it doesn't make sense to have a
		// directory-to-directory transition (although it might later on if
		// we're implementing permissions changes). None of these would stop
		// execution though.

		// Handle the special case where both old and new are a file. In this
		// case we can do a simple swap. It makes sense to handle this specially
		// because it is a very common case and doing it with a swap will remove
		// any window where the path is empty on the filesystem.
		fileToFile := t.Old != nil && t.New != nil &&
			t.Old.Kind == EntryKind_File &&
			t.New.Kind == EntryKind_File
		if fileToFile {
			if err := swap(root, t.Path, t.Old, t.New, cache, provider); err != nil {
				results = append(results, Change{Path: t.Path, New: t.Old})
				problems = append(problems, Problem{Path: t.Path, Error: err.Error()})
			} else {
				results = append(results, Change{Path: t.Path, New: t.New})
			}
			continue
		}

		// If the old entry is non-nil, then transition it to nil (remove it).
		// If this fails, record the reduced entry and the error preventing full
		// deletion.
		if t.Old != nil {
			if reduced, err := remove(root, t.Path, t.Old, cache); err != nil {
				results = append(results, Change{Path: t.Path, New: reduced})
				problems = append(problems, Problem{Path: t.Path, Error: err.Error()})
				continue
			}
		}

		// At this point, we should have nil on disk. Transition to whatever the
		// new entry is. Record whatever portion of the target we can created,
		// and any error preventing full creation.
		created, err := create(root, t.Path, t.New, provider)
		results = append(results, Change{Path: t.Path, New: created})
		if err != nil {
			problems = append(problems, Problem{Path: t.Path, Error: err.Error()})
		}
	}

	// Done.
	return results, problems
}
