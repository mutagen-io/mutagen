package filesystem

import (
	"testing"
)

// TestModePermissionMaskIsExpected is a sanity check that ModePermissionsMask
// is equivalent to 0777 on all platforms (which it should be on POSIX platforms
// under the POSIX standard and on Windows platforms based on the os package's
// (immutable) FileMode definition).
func TestModePermissionMaskIsExpected(t *testing.T) {
	if ModePermissionsMask != Mode(0777) {
		t.Error("ModePermissionsMask value not equal to expected:", ModePermissionsMask, "!=", Mode(0777))
	}
}

// TestModePermissionMaskIsUnionOfPermissions is a sanity check that
// ModePermissionMask is equal to the union of individual permissions.
func TestModePermissionMaskIsUnionOfPermissions(t *testing.T) {
	permissionUnion := ModePermissionUserRead | ModePermissionUserWrite | ModePermissionUserExecute |
		ModePermissionGroupRead | ModePermissionGroupWrite | ModePermissionGroupExecute |
		ModePermissionOthersRead | ModePermissionOthersWrite | ModePermissionOthersExecute
	if ModePermissionsMask != permissionUnion {
		t.Error("ModePermissionsMask value not equal to union of permissions:", ModePermissionsMask, "!=", permissionUnion)
	}
}

// parseModeTestCase represents a test case for ParseMode.
type parseModeTestCase struct {
	// value is the value to parse.
	value string
	// mask is the mask to use in parsing.
	mask Mode
	// expectFailure indicates whether or not parsing failure is expected.
	expectFailure bool
	// expected indicates the expected result in the absence of failure.
	expected Mode
}

// run executes the test in the provided test context.
func (c *parseModeTestCase) run(t *testing.T) {
	// Mark ourselves as a helper function.
	t.Helper()

	// Perform parsing and verify that the expected behavior is observed.
	if result, err := parseMode(c.value, c.mask); err == nil && c.expectFailure {
		t.Fatal("parsing succeeded when failure was expected")
	} else if err != nil && !c.expectFailure {
		t.Fatal("parsing failed unexpectedly:", err)
	} else if result != c.expected {
		t.Error("parsing result does not match expected:", result, "!=", c.expected)
	}
}

// TestParseModeEmpty verifies that parseMode fails on an empty string.
func TestParseModeEmpty(t *testing.T) {
	// Create the test case.
	testCase := &parseModeTestCase{
		mask:          ModePermissionsMask,
		expectFailure: true,
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseModeInvalid verifies that parseMode fails on an invalid string.
func TestParseModeInvalid(t *testing.T) {
	// Create the test case.
	testCase := &parseModeTestCase{
		value:         "laksjfd",
		mask:          ModePermissionsMask,
		expectFailure: true,
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseModeOverflow verifies that parseMode fails on a mode specification
// that overflows an unsigned 32-bit integer.
func TestParseModeOverflow(t *testing.T) {
	// Create the test case.
	testCase := &parseModeTestCase{
		value:         "45201371000",
		mask:          ModePermissionsMask,
		expectFailure: true,
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseModeInvalidBits verifies that parseMode fails on a mode that doesn't
// fit within to the specified bit mask.
func TestParseModeInvalidBits(t *testing.T) {
	// Create the test case.
	testCase := &parseModeTestCase{
		value:         "1000",
		mask:          ModePermissionsMask,
		expectFailure: true,
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseModeValid verifies that parseMode succeeds on a valid string.
func TestParseModeValid(t *testing.T) {
	// Create the test case.
	testCase := &parseModeTestCase{
		value:    "777",
		mask:     ModePermissionsMask,
		expected: 0777,
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseModeValidWithZeroPrefix verifies that parseMode succeeds on a valid
// string with a zero prefix
func TestParseModeValidWithZeroPrefix(t *testing.T) {
	// Create the test case.
	testCase := &parseModeTestCase{
		value:    "0755",
		mask:     ModePermissionsMask,
		expected: 0755,
	}

	// Run the test case.
	testCase.run(t)
}

// TestParseModeValidWithMultiZeroPrefix verifies that parseMode succeeds on a
// valid string with a multi-zero prefix
func TestParseModeValidWithMultiZeroPrefix(t *testing.T) {
	// Create the test case.
	testCase := &parseModeTestCase{
		value:    "00644",
		mask:     ModePermissionsMask,
		expected: 0644,
	}

	// Run the test case.
	testCase.run(t)
}

// TestModeUnmarshalTextUnmodifiedOnFailure verifies that Mode.UnmarshalText
// leaves the underlying mode unmodified in the case of failure.
func TestModeUnmarshalTextUnmodifiedOnFailure(t *testing.T) {
	// Create a zero-valued mode.
	var mode Mode

	// Unmarshal an invalid value.
	if mode.UnmarshalText([]byte("0888")) == nil {
		t.Fatal("mode unmarshalling succeeded unexpectedly")
	} else if mode != 0 {
		t.Error("mode modified during unsuccessful unmarshalling operation")
	}
}
