package core

import (
	"testing"
)

// TestPermissionsModeIsDefault tests PermissionsMode.IsDefault.
func TestPermissionsModeIsDefault(t *testing.T) {
	// Define test cases.
	tests := []struct {
		value    PermissionsMode
		expected bool
	}{
		{PermissionsMode_PermissionsModeDefault - 1, false},
		{PermissionsMode_PermissionsModeDefault, true},
		{PermissionsMode_PermissionsModePortable, false},
		{PermissionsMode_PermissionsModeManual, false},
		{PermissionsMode_PermissionsModeManual + 1, false},
	}

	// Process test cases.
	for i, test := range tests {
		if result := test.value.IsDefault(); result && !test.expected {
			t.Errorf("test index %d: value was unexpectedly classified as default", i)
		} else if !result && test.expected {
			t.Errorf("test index %d: value was unexpectedly classified as non-default", i)
		}
	}
}

// TestPermissionsModeUnmarshalText tests PermissionsMode.UnmarshalText.
func TestPermissionsModeUnmarshalText(t *testing.T) {
	// Define test cases.
	tests := []struct {
		text          string
		expectedMode  PermissionsMode
		expectFailure bool
	}{
		{"", PermissionsMode_PermissionsModeDefault, true},
		{"asdf", PermissionsMode_PermissionsModeDefault, true},
		{"portable", PermissionsMode_PermissionsModePortable, false},
		{"manual", PermissionsMode_PermissionsModeManual, false},
	}

	// Process test cases.
	for _, test := range tests {
		var mode PermissionsMode
		if err := mode.UnmarshalText([]byte(test.text)); err != nil {
			if !test.expectFailure {
				t.Errorf("unable to unmarshal text (%s): %s", test.text, err)
			}
		} else if test.expectFailure {
			t.Error("unmarshaling succeeded unexpectedly for text:", test.text)
		} else if mode != test.expectedMode {
			t.Errorf(
				"unmarshaled mode (%s) does not match expected (%s)",
				mode,
				test.expectedMode,
			)
		}
	}
}

// TestPermissionsModeSupported tests PermissionsMode.Supported.
func TestPermissionsModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode            PermissionsMode
		expectSupported bool
	}{
		{PermissionsMode_PermissionsModeDefault, false},
		{PermissionsMode_PermissionsModePortable, true},
		{PermissionsMode_PermissionsModeManual, true},
		{(PermissionsMode_PermissionsModeManual + 1), false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if supported := testCase.mode.Supported(); supported != testCase.expectSupported {
			t.Errorf(
				"mode support status (%t) does not match expected (%t)",
				supported,
				testCase.expectSupported,
			)
		}
	}
}

// TestPermissionsModeDescription tests PermissionsMode.Description.
func TestPermissionsModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode                PermissionsMode
		expectedDescription string
	}{
		{PermissionsMode_PermissionsModeDefault, "Default"},
		{PermissionsMode_PermissionsModePortable, "Portable"},
		{PermissionsMode_PermissionsModeManual, "Manual"},
		{(PermissionsMode_PermissionsModeManual + 1), "Unknown"},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if description := testCase.mode.Description(); description != testCase.expectedDescription {
			t.Errorf(
				"mode description (%s) does not match expected (%s)",
				description,
				testCase.expectedDescription,
			)
		}
	}
}
