package sync

import (
	"strings"

	"github.com/pkg/errors"
)

// Apply applies a series of changes to a base entry. This function ignores the
// Old value for changes and it assumes all changes are valid to apply against
// the base.
func Apply(base *Entry, changes []*Change) (*Entry, error) {
	// Create a mutable copy of base.
	result := base.Copy()

	// Apply changes.
	for _, c := range changes {
		// Handle the special case of a root path.
		if c.Path == "" {
			result = c.New
			continue
		}

		// Crawl down the tree until we reach the parent of the target location.
		parent := result
		components := strings.Split(c.Path, "/")
		for len(components) > 1 {
			child, ok := parent.Contents[components[0]]
			if !ok {
				return nil, errors.New("unable to resolve parent path")
			}
			parent = child
			components = components[1:]
		}

		// Depending on the new value, either set or remove the entry.
		if c.New == nil {
			delete(parent.Contents, components[0])
		} else {
			if parent.Contents == nil {
				parent.Contents = make(map[string]*Entry)
			}
			parent.Contents[components[0]] = c.New
		}
	}

	// Done.
	return result, nil
}
