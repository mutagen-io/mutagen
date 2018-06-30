package session

import (
	"github.com/havoc-io/mutagen/pkg/sync"
)

func isRootDeletion(change *sync.Change) bool {
	return change.Path == "" && change.Old != nil && change.New == nil
}

func isRootTypeChange(change *sync.Change) bool {
	return change.Path == "" &&
		change.Old != nil && change.New != nil &&
		change.Old.Kind != change.New.Kind
}
