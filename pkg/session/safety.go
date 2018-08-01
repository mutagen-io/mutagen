package session

import (
	"github.com/havoc-io/mutagen/pkg/sync"
)

// isRootDeletion determines whether or not the specified change is a root
// deletion.
func isRootDeletion(change *sync.Change) bool {
	return change.Path == "" && change.Old != nil && change.New == nil
}

// isRootTypeChange determines whether or not the specified change is a root
// type change.
func isRootTypeChange(change *sync.Change) bool {
	return change.Path == "" &&
		change.Old != nil && change.New != nil &&
		change.Old.Kind != change.New.Kind
}
