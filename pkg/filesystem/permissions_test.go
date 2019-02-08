package filesystem

import (
	"testing"
)

// parseOwnershipIdentifierTestCase represents a test case for
// ParseOwnershipIdentifier.
type parseOwnershipIdentifierTestCase struct {
	// specification is the ownership specification.
	specification string
	// expectedKind is the expected OwnershipIdentifierKind.
	expectedKind OwnershipIdentifierKind
	// expectedValue is the expected ownership specification value.
	expectedValue string
}

// run executes the test in the provided test context.
func (c *parseOwnershipIdentifierTestCase) run(t *testing.T) {
	// Mark ourselves as a helper function.
	t.Helper()

	// Perform the parsing.
	kind, value := ParseOwnershipIdentifier(c.specification)

	// Check results.
	if kind != c.expectedKind {
		t.Error("parsed kind does not match expected:", kind, "!=", c.expectedKind)
	}
	if value != c.expectedValue {
		t.Error("parsed value does not match expected:", value, "!=", c.expectedValue)
	}
}

// TestParseOwnershipIdentifierEmpty tests that parsing of an empty ownership
// specification fails.
func TestParseOwnershipIdentifierEmpty(t *testing.T) {
	// Create the test case.
	testCase := &parseOwnershipIdentifierTestCase{
		specification: "",
		expectedKind:  OwnershipIdentifierKindInvalid,
		expectedValue: "",
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseOwnershipIdentifierPOSIXIDEmpty tests that parsing of an empty POSIX
// ID specification fails.
func TestParseOwnershipIdentifierPOSIXIDEmpty(t *testing.T) {
	// Create the test case.
	testCase := &parseOwnershipIdentifierTestCase{
		specification: "id:",
		expectedKind:  OwnershipIdentifierKindInvalid,
		expectedValue: "",
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseOwnershipIdentifierPOSIXIDOctal tests that parsing of an octal POSIX
// ID specification fails.
func TestParseOwnershipIdentifierPOSIXIDOctal(t *testing.T) {
	// Create the test case.
	testCase := &parseOwnershipIdentifierTestCase{
		specification: "id:0442",
		expectedKind:  OwnershipIdentifierKindInvalid,
		expectedValue: "",
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseOwnershipIdentifierPOSIXIDAlphaNumeric tests that parsing of an
// alphanumeric POSIX ID specification fails.
func TestParseOwnershipIdentifierPOSIXIDAlphaNumeric(t *testing.T) {
	// Create the test case.
	testCase := &parseOwnershipIdentifierTestCase{
		specification: "id:5a42",
		expectedKind:  OwnershipIdentifierKindInvalid,
		expectedValue: "",
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseOwnershipIdentifierPOSIXIDRoot tests that parsing of a root POSIX ID
// specification succeeds.
func TestParseOwnershipIdentifierPOSIXIDRoot(t *testing.T) {
	// Create the test case.
	testCase := &parseOwnershipIdentifierTestCase{
		specification: "id:0",
		expectedKind:  OwnershipIdentifierKindPOSIXID,
		expectedValue: "0",
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseOwnershipIdentifierPOSIXIDSingleDigit tests that parsing of a
// single-digit POSIX ID specification succeeds.
func TestParseOwnershipIdentifierPOSIXIDSingleDigit(t *testing.T) {
	// Create the test case.
	testCase := &parseOwnershipIdentifierTestCase{
		specification: "id:4",
		expectedKind:  OwnershipIdentifierKindPOSIXID,
		expectedValue: "4",
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseOwnershipIdentifierPOSIXIDMultiDigit tests that parsing of a
// multi-digit POSIX ID specification succeeds.
func TestParseOwnershipIdentifierPOSIXIDMultiDigit(t *testing.T) {
	// Create the test case.
	testCase := &parseOwnershipIdentifierTestCase{
		specification: "id:454",
		expectedKind:  OwnershipIdentifierKindPOSIXID,
		expectedValue: "454",
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseOwnershipIdentifierWindowsSIDEmpty tests that parsing of an empty
// Windows SID specification fails.
func TestParseOwnershipIdentifierWindowsSIDEmpty(t *testing.T) {
	// Create the test case.
	testCase := &parseOwnershipIdentifierTestCase{
		specification: "sid:",
		expectedKind:  OwnershipIdentifierKindInvalid,
		expectedValue: "",
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseOwnershipIdentifierWindowsSIDStringConstant tests that parsing of a
// string-constant-based Windows SID specification succeeds.
func TestParseOwnershipIdentifierWindowsSIDStringConstant(t *testing.T) {
	// Create the test case.
	testCase := &parseOwnershipIdentifierTestCase{
		specification: "sid:BA",
		expectedKind:  OwnershipIdentifierKindWindowsSID,
		expectedValue: "BA",
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseOwnershipIdentifierWindowsSIDWellKnown tests that parsing of a
// well-known Windows SID specification succeeds.
func TestParseOwnershipIdentifierWindowsSIDWellKnown(t *testing.T) {
	// Create the test case.
	testCase := &parseOwnershipIdentifierTestCase{
		specification: "sid:S-1-3-0",
		expectedKind:  OwnershipIdentifierKindWindowsSID,
		expectedValue: "S-1-3-0",
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseOwnershipIdentifierName tests that parsing of a name specification
// succeeds.
func TestParseOwnershipIdentifierName(t *testing.T) {
	// Create the test case.
	testCase := &parseOwnershipIdentifierTestCase{
		specification: "george",
		expectedKind:  OwnershipIdentifierKindName,
		expectedValue: "george",
	}

	// Run the test case.
	testCase.run(t)
}
