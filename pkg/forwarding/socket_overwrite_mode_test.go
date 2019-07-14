package forwarding

import (
	"testing"
)

// TestSocketOverwriteModeUnmarshal tests that unmarshaling from a string
// specification succeeeds for SocketOverwriteMode.
func TestSocketOverwriteModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		text          string
		expectedMode  SocketOverwriteMode
		expectFailure bool
	}{
		{"", SocketOverwriteMode_SocketOverwriteModeDefault, true},
		{"asdf", SocketOverwriteMode_SocketOverwriteModeDefault, true},
		{"leave", SocketOverwriteMode_SocketOverwriteModeLeave, false},
		{"overwrite", SocketOverwriteMode_SocketOverwriteModeOverwrite, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode SocketOverwriteMode
		if err := mode.UnmarshalText([]byte(testCase.text)); err != nil {
			if !testCase.expectFailure {
				t.Errorf("unable to unmarshal text (%s): %s", testCase.text, err)
			}
		} else if testCase.expectFailure {
			t.Error("unmarshaling succeeded unexpectedly for text:", testCase.text)
		} else if mode != testCase.expectedMode {
			t.Errorf(
				"unmarshaled mode (%s) does not match expected (%s)",
				mode,
				testCase.expectedMode,
			)
		}
	}
}

// TestSocketOverwriteModeSupported tests that SocketOverwriteMode support
// detection works as expected.
func TestSocketOverwriteModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode            SocketOverwriteMode
		expectSupported bool
	}{
		{SocketOverwriteMode_SocketOverwriteModeDefault, false},
		{SocketOverwriteMode_SocketOverwriteModeLeave, true},
		{SocketOverwriteMode_SocketOverwriteModeOverwrite, true},
		{(SocketOverwriteMode_SocketOverwriteModeOverwrite + 1), false},
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

// TestSocketOverwriteModeDescription tests that SocketOverwriteMode description
// generation works as expected.
func TestSocketOverwriteModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode                SocketOverwriteMode
		expectedDescription string
	}{
		{SocketOverwriteMode_SocketOverwriteModeDefault, "Default"},
		{SocketOverwriteMode_SocketOverwriteModeLeave, "Leave"},
		{SocketOverwriteMode_SocketOverwriteModeOverwrite, "Overwrite"},
		{(SocketOverwriteMode_SocketOverwriteModeOverwrite + 1), "Unknown"},
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
