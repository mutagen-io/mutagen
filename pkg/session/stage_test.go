package session

import (
	"testing"
)

// TestStageModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for StageMode.
func TestStageModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Text          string
		ExpectedMode  StageMode
		ExpectFailure bool
	}{
		{"", StageMode_StageModeDefault, true},
		{"asdf", StageMode_StageModeDefault, true},
		{"mutagen", StageMode_StageModeMutagen, false},
		{"neighboring", StageMode_StageModeNeighboring, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode StageMode
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

// TestStageModeSupported tests that StageMode support detection works as
// expected.
func TestStageModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode            StageMode
		ExpectSupported bool
	}{
		{StageMode_StageModeDefault, false},
		{StageMode_StageModeMutagen, true},
		{StageMode_StageModeNeighboring, true},
		{(StageMode_StageModeNeighboring + 1), false},
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

// TestStageModeDescription tests that StageMode description generation works as
// expected.
func TestStageModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode                StageMode
		ExpectedDescription string
	}{
		{StageMode_StageModeDefault, "Default"},
		{StageMode_StageModeMutagen, "Mutagen"},
		{StageMode_StageModeNeighboring, "Neighboring"},
		{(StageMode_StageModeNeighboring + 1), "Unknown"},
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
