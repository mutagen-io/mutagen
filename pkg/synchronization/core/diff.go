package core

// differ provides recursive diffing infrastructure.
type differ struct {
	// changes is the list of changes being tracked by the diff.
	changes []*Change
}

// diff is the recursive diff entry point.
func (d *differ) diff(path string, base, target *Entry) {
	// If the nodes at this path aren't equal, then do a complete replacement.
	if !target.Equal(base, false) {
		d.changes = append(d.changes, &Change{
			Path: path,
			Old:  base,
			New:  target,
		})
		return
	}

	// Otherwise check contents for differences.
	baseContents := base.GetContents()
	targetContents := target.GetContents()
	for name := range nameUnion(baseContents, targetContents) {
		d.diff(pathJoin(path, name), baseContents[name], targetContents[name])
	}
}

// diff performs a diff operation between a base and target entry (treating both
// as rooted at the specified path) and generates a list of changes that, if
// applied to base, would transform it into target.
func diff(path string, base, target *Entry) []*Change {
	// Create the differ.
	d := &differ{}

	// Populate changes.
	d.diff(path, base, target)

	// Done.
	return d.changes
}

// Diff performs a diff operation between a base and target entry and generates
// a list of changes that, if applied to base, would transform it into target.
func Diff(base, target *Entry) []*Change {
	return diff("", base, target)
}
