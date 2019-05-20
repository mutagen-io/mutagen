package session

import (
	"testing"
)

// TestWatchModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for WatchMode.
func TestWatchModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		text          string
		expectedMode  WatchMode
		expectFailure bool
	}{
		{"", WatchMode_WatchModeDefault, true},
		{"asdf", WatchMode_WatchModeDefault, true},
		{"portable", WatchMode_WatchModePortable, false},
		{"force-poll", WatchMode_WatchModeForcePoll, false},
		{"no-watch", WatchMode_WatchModeNoWatch, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode WatchMode
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

// TestWatchModeSupported tests that WatchMode support detection works as
// expected.
func TestWatchModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode            WatchMode
		expectSupported bool
	}{
		{WatchMode_WatchModeDefault, false},
		{WatchMode_WatchModePortable, true},
		{WatchMode_WatchModeForcePoll, true},
		{WatchMode_WatchModeNoWatch, true},
		{(WatchMode_WatchModeNoWatch + 1), false},
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

// TestWatchModeDescription tests that WatchMode description generation works as
// expected.
func TestWatchModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode                WatchMode
		expectedDescription string
	}{
		{WatchMode_WatchModeDefault, "Default"},
		{WatchMode_WatchModePortable, "Portable"},
		{WatchMode_WatchModeForcePoll, "Force Poll"},
		{WatchMode_WatchModeNoWatch, "No Watch"},
		{(WatchMode_WatchModeNoWatch + 1), "Unknown"},
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
