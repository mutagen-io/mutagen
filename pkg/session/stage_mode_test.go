package session

import (
	"testing"
)

// TestStageModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for StageMode.
func TestStageModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		text          string
		expectedMode  StageMode
		expectFailure bool
	}{
		{"", StageMode_StageModeDefault, true},
		{"asdf", StageMode_StageModeDefault, true},
		{"mutagen", StageMode_StageModeMutagen, false},
		{"neighboring", StageMode_StageModeNeighboring, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode StageMode
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

// TestStageModeSupported tests that StageMode support detection works as
// expected.
func TestStageModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode            StageMode
		expectSupported bool
	}{
		{StageMode_StageModeDefault, false},
		{StageMode_StageModeMutagen, true},
		{StageMode_StageModeNeighboring, true},
		{(StageMode_StageModeNeighboring + 1), false},
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

// TestStageModeDescription tests that StageMode description generation works as
// expected.
func TestStageModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode                StageMode
		expectedDescription string
	}{
		{StageMode_StageModeDefault, "Default"},
		{StageMode_StageModeMutagen, "Mutagen Data Directory"},
		{StageMode_StageModeNeighboring, "Neighboring"},
		{(StageMode_StageModeNeighboring + 1), "Unknown"},
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
