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
	// Set up test cases.
	testCases := []string{
		PrefixSynchronization,
		PrefixForwarding,
		PrefixProject,
	}

	// Process test cases.
	for _, prefix := range testCases {
		// Create an identifier with the specified prefix.
		identifier, err := New(prefix)
		if err != nil {
			t.Fatal("unable to create identifier:", err)
		}

		// Ensure that the prefix is present.
		if !strings.HasPrefix(identifier, prefix) {
			t.Error("identifier does not have correct prefix")
		}

		// Ensure that the length is what's expected.
		if len(identifier) != expectedIdentifierLength {
			t.Error("identifier has unexpected length")
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
	// Set up test cases.
	testCases := []struct {
		value       string
		expectValid bool
	}{
		{"", false},
		{"abc", false},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"75A0FDC4-5C08-4AA4-99B5-154350DEA3DB", false},
		{"75a0fdc4-5c08-4aa4-99b5-154350dea3dba", false},
		{"proj_jndACgB0qejgkorhU21q4oA56QvEfqV1p2yBH9N40h+", false},
		{"proj_jndACgB0qejgkorhU21q4oA56QvEfqV1p2yBH9N40hK1", false},
		{"pro9_jndACgB0qejgkorhU21q4oA56QvEfqV1p2yBH9N40hK", false},
		{"PROJ_jndACgB0qejgkorhU21q4oA56QvEfqV1p2yBH9N40hK", false},
		{"75a0fdc4-5c08-4aa4-99b5-154350dea3db", true},
		{"proj_jndACgB0qejgkorhU21q4oA56QvEfqV1p2yBH9N40hK", true},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if valid := IsValid(testCase.value); valid && !testCase.expectValid {
			t.Error("identifier unexpectedly classified as valid:", testCase.value)
		} else if !valid && testCase.expectValid {
			t.Error("identifier unexpectedly classified as invalid:", testCase.value)
		}
	}
}
