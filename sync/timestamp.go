package sync

import (
	"github.com/golang/protobuf/ptypes/timestamp"
)

func timestampsEqual(first, second *timestamp.Timestamp) bool {
	// If both timestamps are nil, we can't perform any sensible comparison, so
	// err on the the side of caution.
	if first == nil && second == nil {
		return false
	}

	// If only one is nil, they obviously aren't equal.
	if first == nil {
		return false
	} else if second == nil {
		return false
	}

	// Otherwise compare fields.
	return first.Seconds == second.Seconds && first.Nanos == second.Nanos
}
