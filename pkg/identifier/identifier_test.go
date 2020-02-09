package identifier

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/encoding"
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
		PrefixPrompter,
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

// TestLeftPadRemoval tests that the original bytes of an identifier can be
// extracted after padding in Base62 encoding.
func TestLeftPadRemoval(t *testing.T) {
	// Set up test cases. We use 16 byte values, which means that the target
	// length for Base62-encoded values should be 22.
	testCases := [][]byte{
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
		{0x01, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
		{0xf2, 0xa7, 0x30, 0x90, 0x01, 0x7b, 0x00, 0x01, 0xff, 0xfe, 0x0f, 0x1f, 0xa1, 0x0a, 0x0f, 0xf0},
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	}

	// Process test cases.
	for _, value := range testCases {
		// Encode the value.
		encoded := encoding.EncodeBase62(value)

		// Create a string builder.
		builder := &strings.Builder{}

		// If the encoded value has a length less than the target length, then
		// left-pad it with 0s.
		for i := 22 - len(encoded); i > 0; i-- {
			builder.WriteByte(encoding.Base62Alphabet[0])
		}

		// Write the encoded value.
		builder.WriteString(encoded)

		// Decode the resulting string.
		decoded, err := encoding.DecodeBase62(builder.String())
		if err != nil {
			t.Error("unable to decode value:", err)
		} else if !bytes.Equal(decoded[len(decoded)-16:], value) {
			t.Error("decoded and extracted bytes do not match original")
		}
	}
}
