package sync

import (
	pathpkg "path"
)

type differ struct {
	changes []*Change
}

func (d *differ) diff(path string, base, target *Entry) {
	// Handle nil cases.
	if base == nil && target == nil {
		return
	}

	// If the nodes at this path aren't equal, then do a complete replacement.
	if !target.equalShallow(base) {
		d.changes = append(d.changes, &Change{Path: path, Old: base, New: target})
		return
	}

	// Otherwise check contents for differences.
	for n, _ := range iterate(base.Contents, target.Contents) {
		d.diff(pathpkg.Join(path, n), base.Contents[n], target.Contents[n])
	}
}

func Diff(base, target *Entry) []*Change {
	// Create the differ.
	d := &differ{}

	// Populate changes.
	d.diff("", base, target)

	// Done.
	return d.changes
}
