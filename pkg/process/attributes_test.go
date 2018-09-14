package process

import (
	"testing"
)

func TestDetachedProcessAttributes(t *testing.T) {
	if DetachedProcessAttributes() == nil {
		t.Error("nil detached process attributes returned")
	}
}
