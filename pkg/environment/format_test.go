package environment

import (
	"testing"
)

func TestFormat(t *testing.T) {
	// Create a copy of the current environment.
	preformat := CopyCurrent()

	// Add some test environment variables.
	preformat["WASHINGTON"] = "George"
	preformat["ADAMS"] = "John\nQuincy"
	preformat["Jefferson"] = "Thomas"

	// Format this copy.
	formatted := Format(preformat)

	// Reparse the copy.
	reparsed, err := Parse(formatted)
	if err != nil {
		t.Fatal("unable to reparse formatted environment")
	}

	// Ensure that it has the same size as the original.
	if len(reparsed) != len(preformat) {
		t.Error("reparsed environment length does not match pre-format environment length")
	}

	// Ensure that it has the same contents as the original.
	for k, rv := range reparsed {
		if pv, ok := preformat[k]; !ok {
			t.Error("reparsed environment has extra key:", k)
		} else if rv != pv {
			t.Error("reparsed environment value doesn't match original:", rv, "!=", pv)
		}
	}
}
