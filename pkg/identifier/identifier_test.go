package identifier

import (
	"math"
	"strings"
	"testing"
)

const (
	// expectedIdentifierLength is the expected length for identifiers.
	expectedIdentifierLength = requiredPrefixLength + 1 + targetBase62Length
)

// TestLengthRelationships tests the mathematical relationship between
// collisionResistantLength and targetBase62Length.
func TestLengthRelationships(t *testing.T) {
	if targetBase62Length != int(math.Ceil(collisionResistantLength*8*math.Log(2)/math.Log(62))) {
		t.Error("target base62 length incorrect for collision resistant length")
	}
}

// TestIdentifierCreation tests identifier creation.
func TestIdentifierCreation(t *testing.T) {
	// Define test cases.
	tests := []string{
		PrefixSynchronization,
		PrefixForwarding,
		PrefixProject,
		PrefixPrompter,
		PrefixToken,
	}

	// Process test cases.
	for _, prefix := range tests {
		// Create an identifier with the specified prefix.
		identifier, err := New(prefix)
		if err != nil {
			t.Fatalf("unable to create identifier with prefix (%s): %v", prefix, err)
		}

		// Ensure that the prefix is present.
		if !strings.HasPrefix(identifier, prefix) {
			t.Errorf("identifier (%s) does not have correct prefix (%s)", identifier, prefix)
		}

		// Ensure that the length is what's expected.
		if len(identifier) != expectedIdentifierLength {
			t.Errorf("identifier (%s) has unexpected length: %d != %d",
				identifier, len(identifier), expectedIdentifierLength,
			)
		}
	}
}

// TestInvalidPrefixLength tests that identifier creation fails with an invalid
// prefix length.
func TestPrefixLengthEnforcement(t *testing.T) {
	if _, err := New("xyz"); err == nil {
		t.Error("invalid prefix length accepted")
	}
}

// TestInvalidPrefixCharacter tests that identifier creation fails when a prefix
// contains invalid characters.
func TestInvalidPrefixCharacter(t *testing.T) {
	if _, err := New("XYZ"); err == nil {
		t.Error("invalid prefix characters accepted")
	}
}

// TestIsValid tests that IsValid behaves correctly for an assortment of values.
func TestIsValid(t *testing.T) {
	// Define test cases.
	tests := []struct {
		value       string
		allowLegacy bool
		expectValid bool
	}{
		{"", true, false},
		{"abc", true, false},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true, false},
		{"75A0FDC4-5C08-4AA4-99B5-154350DEA3DB", true, false},
		{"75a0fdc4-5c08-4aa4-99b5-154350dea3dba", true, false},
		{"proj_jndACgB0qejgkorhU21q4oA56QvEfqV1p2yBH9N40h+", true, false},
		{"proj_jndACgB0qejgkorhU21q4oA56QvEfqV1p2yBH9N40hK1", true, false},
		{"pro9_jndACgB0qejgkorhU21q4oA56QvEfqV1p2yBH9N40hK", true, false},
		{"PROJ_jndACgB0qejgkorhU21q4oA56QvEfqV1p2yBH9N40hK", true, false},
		{"75a0fdc4-5c08-4aa4-99b5-154350dea3db", false, false},
		{"75a0fdc4-5c08-4aa4-99b5-154350dea3db", true, true},
		{"proj_jndACgB0qejgkorhU21q4oA56QvEfqV1p2yBH9N40hK", false, true},
		{"proj_jndACgB0qejgkorhU21q4oA56QvEfqV1p2yBH9N40hK", true, true},
	}

	// Process test cases.
	for _, test := range tests {
		legacy := "legacy not allowed"
		if test.allowLegacy {
			legacy = "legacy allowed"
		}
		if valid := IsValid(test.value, test.allowLegacy); valid && !test.expectValid {
			t.Errorf("identifier (%s) unexpectedly classified as valid (%s)",
				test.value, legacy,
			)
		} else if !valid && test.expectValid {
			t.Errorf("identifier (%s) unexpectedly classified as invalid (%s)",
				test.value, legacy,
			)
		}
	}
}
