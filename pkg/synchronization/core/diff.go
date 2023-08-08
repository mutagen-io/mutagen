package core

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/fastpath"
)

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

	// Extract contents.
	baseContents := base.GetContents()
	targetContents := target.GetContents()

	// Compute the prefix to add to content names to compute their paths.
	var contentPathPrefix string
	if len(baseContents) > 0 || len(targetContents) > 0 {
		contentPathPrefix = fastpath.Joinable(path)
	}

	// The nodes were equal at this path, so check their contents.
	for name := range nameUnion(baseContents, targetContents) {
		d.diff(contentPathPrefix+name, baseContents[name], targetContents[name])
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
