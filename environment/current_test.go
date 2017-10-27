package environment

import (
	"os"

	"testing"
)

func TestCurrent(t *testing.T) {
	// Ensure that our copy of the environment matches the length of the Go
	// runtime's.
	if len(Current) != len(os.Environ()) {
		t.Error("parsed environment does not match native length")
	}

	// Ensure that our version of the environment matches what's in the Go
	// runtime's environment.
	for k, cv := range Current {
		if ov := os.Getenv(k); cv != ov {
			t.Error("parsed environment value doesn't match original:", cv, "!=", ov)
		}
	}
}

func TestCopyCurrent(t *testing.T) {
	// Create a copy of the current environment.
	duplicated := CopyCurrent()

	// Ensure that it has the same size as the original.
	if len(duplicated) != len(Current) {
		t.Error("duplicated environment does not match original length")
	}

	// Ensure that it has the same contents as the original.
	for k, dv := range duplicated {
		if cv, ok := Current[k]; !ok {
			t.Error("duplicated environment has extra key:", k)
		} else if dv != cv {
			t.Error("duplicated environment value doesn't match original:", dv, "!=", cv)
		}
	}
}
