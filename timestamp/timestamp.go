package timestamp

import (
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func Convert(t time.Time) (*timestamp.Timestamp, error) {
	return ptypes.TimestampProto(t)
}

func Equal(first, second *timestamp.Timestamp) bool {
	// If one or both of the timestamps is nil, we can't perform any sensible
	// comparison, so err on the the side of caution.
	if first == nil || second == nil {
		return false
	}

	// Otherwise compare fields. Protocol Buffers timestamps don't allow
	// negative values for nanoseconds, so any time that has a representation
	// has a unique representation.
	return first.Seconds == second.Seconds && first.Nanos == second.Nanos
}

func Less(first, second *timestamp.Timestamp) bool {
	// If one or both of the timestamps is nil, we can't perform any sensible
	// comparison, so err on the the side of caution.
	if first == nil || second == nil {
		return false
	}

	// Compare components. This comparison relies on the fact that Nanos can't
	// be negative (at least not according to the Protocol Buffers definition of
	// its value) and its value is restricted to the range [0, 999,999,999].
	// Without these conditions we'd have to perform a normalization pass first.
	return first.Seconds < second.Seconds ||
		(first.Seconds == second.Seconds && first.Nanos < second.Nanos)
}
