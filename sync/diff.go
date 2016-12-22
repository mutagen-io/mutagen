package sync

import (
	pathpkg "path"
)

type differ struct {
	changes []Change
}

func (d *differ) diff(path string, base, target *Entry) {
	// If the nodes at this path aren't equal, then do a complete replacement.
	if !target.equalShallow(base) {
		d.changes = append(d.changes, Change{
			Path: path,
			Old:  base,
			New:  target,
		})
		return
	}

	// Otherwise check contents for differences.
	baseContents := base.GetContents()
	targetContents := target.GetContents()
	for name, _ := range nameUnion(baseContents, targetContents) {
		d.diff(pathpkg.Join(path, name), baseContents[name], targetContents[name])
	}
}

func diff(path string, base, target *Entry) []Change {
	// Create the differ.
	d := &differ{}

	// Populate changes.
	d.diff(path, base, target)

	// Done.
	return d.changes
}
