package core

import (
	"errors"
	"strings"
)

// Apply applies a series of changes to a base entry. It ignores the Old value
// for changes and only fails if the path to a change can't be resolved.
func Apply(base *Entry, changes []*Change) (*Entry, error) {
	// If there are no changes, then we can just return the base unmodified.
	if len(changes) == 0 {
		return base, nil
	}

	// If there's only a single change and it's a root replacement, then we can
	// just return the new entry.
	if len(changes) == 1 && changes[0].Path == "" {
		return changes[0].New, nil
	}

	// Create a deep copy of the base entry for mutation.
	result := base.Copy(true)

	// Apply changes.
	for _, change := range changes {
		// Handle the special case of a root replacement. This typically won't
		// occur mid-change-list, so we don't optimize for this case here in the
		// same way that we do above.
		if change.Path == "" {
			result = change.New.Copy(true)
			continue
		}

		// Crawl down the tree until we reach the parent of the target location.
		parent := result
		components := strings.Split(change.Path, "/")
		for len(components) > 1 {
			child, ok := parent.Contents[components[0]]
			if !ok {
				return nil, errors.New("unable to resolve parent path")
			}
			parent = child
			components = components[1:]
		}

		// Depending on the new value, either set or remove the entry. If we're
		// setting a new entry, then we need to create a mutable copy of it in
		// case any subsequent changes affect it.
		if change.New == nil {
			delete(parent.Contents, components[0])
		} else {
			if parent.Contents == nil {
				parent.Contents = make(map[string]*Entry)
			}
			parent.Contents[components[0]] = change.New.Copy(true)
		}
	}

	// Done.
	return result, nil
}
