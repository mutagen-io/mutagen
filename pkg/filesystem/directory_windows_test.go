package filesystem

import (
	"os"
	"testing"
	"unicode/utf8"
)

// TestPathSeparatorsSingleByte verifies that the path separator runes are
// encoded as a single byte. We rely on this assumption for high performance in
// ensureValidName.
func TestPathSeparatorsSingleByte(t *testing.T) {
	if utf8.RuneLen(os.PathSeparator) != 1 {
		t.Fatal("OS path separator does not have single-byte encoding")
	} else if utf8.RuneLen('/') != 1 {
		t.Fatal("alternate OS path separator does not have single-byte encoding")
	}
}
