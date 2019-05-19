package watching

import (
	"testing"
)

// TestWatchModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for WatchMode.
func TestWatchModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Text          string
		ExpectedMode  WatchMode
		ExpectFailure bool
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
		if err := mode.UnmarshalText([]byte(testCase.Text)); err != nil {
			if !testCase.ExpectFailure {
				t.Errorf("unable to unmarshal text (%s): %s", testCase.Text, err)
			}
		} else if testCase.ExpectFailure {
			t.Error("unmarshaling succeeded unexpectedly for text:", testCase.Text)
		} else if mode != testCase.ExpectedMode {
			t.Errorf(
				"unmarshaled mode (%s) does not match expected (%s)",
				mode,
				testCase.ExpectedMode,
			)
		}
	}
}

// TestWatchModeSupported tests that WatchMode support detection works as
// expected.
func TestWatchModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode            WatchMode
		ExpectSupported bool
	}{
		{WatchMode_WatchModeDefault, false},
		{WatchMode_WatchModePortable, true},
		{WatchMode_WatchModeForcePoll, true},
		{WatchMode_WatchModeNoWatch, true},
		{(WatchMode_WatchModeNoWatch + 1), false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if supported := testCase.Mode.Supported(); supported != testCase.ExpectSupported {
			t.Errorf(
				"mode support status (%t) does not match expected (%t)",
				supported,
				testCase.ExpectSupported,
			)
		}
	}
}

// TestWatchModeDescription tests that WatchMode description generation works as
// expected.
func TestWatchModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode                WatchMode
		ExpectedDescription string
	}{
		{WatchMode_WatchModeDefault, "Default"},
		{WatchMode_WatchModePortable, "Portable"},
		{WatchMode_WatchModeForcePoll, "Force Poll"},
		{WatchMode_WatchModeNoWatch, "No Watch"},
		{(WatchMode_WatchModeNoWatch + 1), "Unknown"},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if description := testCase.Mode.Description(); description != testCase.ExpectedDescription {
			t.Errorf(
				"mode description (%s) does not match expected (%s)",
				description,
				testCase.ExpectedDescription,
			)
		}
	}
}
