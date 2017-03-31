package timestamp

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
)

func TestConvertNow(t *testing.T) {
	if timestamp, err := Convert(time.Now()); err != nil {
		t.Fatal("timestamp conversion failed for current time:", err)
	} else if timestamp == nil {
		t.Fatal("timestamp conversion unexpectedly returned nil timestamp")
	}
}

func TestEqualBothNil(t *testing.T) {
	if Equal(nil, nil) {
		t.Fatal("nil timestamps compared equal")
	}
}

func TestEqualFirstNil(t *testing.T) {
	if Equal(nil, &timestamp.Timestamp{0, 0}) {
		t.Fatal("nil timestamp compared equal to non-nil timestamp")
	}
}

func TestEqualSecondNil(t *testing.T) {
	if Equal(&timestamp.Timestamp{0, 0}, nil) {
		t.Fatal("non-nil timestamp compared equal to nil timestamp")
	}
}

func TestEqual(t *testing.T) {
	if !Equal(&timestamp.Timestamp{12, 3456}, &timestamp.Timestamp{12, 3456}) {
		t.Fatal("equal timestamps compared non-equal")
	}
}

func TestLessBothNil(t *testing.T) {
	if Less(nil, nil) {
		t.Fatal("nil timestamps compared less")
	}
}

func TestLessFirstNil(t *testing.T) {
	if Less(nil, &timestamp.Timestamp{0, 0}) {
		t.Fatal("nil timestamp compared less to non-nil timestamp")
	}
}

func TestLessSecondNil(t *testing.T) {
	if Less(&timestamp.Timestamp{0, 0}, nil) {
		t.Fatal("non-nil timestamp compared less to nil timestamp")
	}
}

func TestLessEqual(t *testing.T) {
	if Less(&timestamp.Timestamp{12, 3456}, &timestamp.Timestamp{12, 3456}) {
		t.Fatal("equal timestamps compared less")
	}
}

func TestSecondsLessBothPositive(t *testing.T) {
	if !Less(&timestamp.Timestamp{99, 0}, &timestamp.Timestamp{100, 0}) {
		t.Fatal("timestamp with lower seconds compared not less")
	}
}

func TestSecondsLessFirstNegativeSecondPositive(t *testing.T) {
	if !Less(&timestamp.Timestamp{-1, 0}, &timestamp.Timestamp{1, 0}) {
		t.Fatal("timestamp with lower seconds compared not less")
	}
}

func TestSecondsLessBothNegative(t *testing.T) {
	if !Less(&timestamp.Timestamp{-100, 0}, &timestamp.Timestamp{-99, 0}) {
		t.Fatal("timestamp with lower seconds compared not less")
	}
}

func TestSecondsGreaterBothPositive(t *testing.T) {
	if Less(&timestamp.Timestamp{100, 0}, &timestamp.Timestamp{99, 0}) {
		t.Fatal("timestamp with higher seconds compared less")
	}
}

func TestSecondsGreaterFirstPositiveSecondNegative(t *testing.T) {
	if Less(&timestamp.Timestamp{1, 0}, &timestamp.Timestamp{-1, 0}) {
		t.Fatal("timestamp with higher seconds compared less")
	}
}

func TestSecondsGreaterBothNegative(t *testing.T) {
	if Less(&timestamp.Timestamp{-99, 0}, &timestamp.Timestamp{-100, 0}) {
		t.Fatal("timestamp with higher seconds compared less")
	}
}

func TestNanosecondsLess(t *testing.T) {
	if !Less(&timestamp.Timestamp{0, 99}, &timestamp.Timestamp{0, 100}) {
		t.Fatal("timestamp with lower nanoseconds compared not less")
	}
}

func TestNanosecondsGreater(t *testing.T) {
	if Less(&timestamp.Timestamp{0, 100}, &timestamp.Timestamp{0, 99}) {
		t.Fatal("timestamp with higher nanoseconds compared less")
	}
}
