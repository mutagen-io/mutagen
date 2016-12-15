package sync

import (
	pathpkg "path"
)

type differ struct {
	changes []*Change
}

func (d *differ) diff(path string, base, target *Entry) {
	// If the nodes at this path aren't equal, then do a complete replacement.
	if !target.equalShallow(base) {
		d.changes = append(d.changes, &Change{
			Path: path,
			Old:  base,
			New:  target,
		})
		return
	}

	// Otherwise check contents for differences.
	iterate2(base.GetContents(), target.GetContents(),
		func(name string, b, t *Entry) {
			d.diff(pathpkg.Join(path, name), b, t)
		},
	)
}

func Diff(base, target *Entry) []*Change {
	// Create the differ.
	d := &differ{}

	// Populate changes.
	d.diff("", base, target)

	// Done.
	return d.changes
}
