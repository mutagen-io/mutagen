// +build !windows

package filesystem

import (
	"os"
	"testing"
	"unicode/utf8"
)

// TestPathSeparatorSingleByte verifies that the path separator rune is encoded
// as a single byte. We rely on this assumption for high performance in
// ensureValidName.
func TestPathSeparatorSingleByte(t *testing.T) {
	if utf8.RuneLen(os.PathSeparator) != 1 {
		t.Fatal("OS path separator does not have single-byte encoding")
	}
}

// TODO: Implement additional tests.
