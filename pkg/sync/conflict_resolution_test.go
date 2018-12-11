package sync

import (
	"testing"
)

// TestConflictResolutionModeUnmarshal tests that unmarshaling from a string
// specification succeeeds for ConflictResolutionMode.
func TestConflictResolutionModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Text          string
		ExpectedMode  ConflictResolutionMode
		ExpectFailure bool
	}{
		{"", ConflictResolutionMode_ConflictResolutionModeDefault, true},
		{"asdf", ConflictResolutionMode_ConflictResolutionModeDefault, true},
		{"safe", ConflictResolutionMode_ConflictResolutionModeSafe, false},
		{"alpha-wins", ConflictResolutionMode_ConflictResolutionModeAlphaWins, false},
		{"beta-wins", ConflictResolutionMode_ConflictResolutionModeBetaWins, false},
		{"alpha-wins-all", ConflictResolutionMode_ConflictResolutionModeAlphaWinsAll, false},
		{"beta-wins-all", ConflictResolutionMode_ConflictResolutionModeBetaWinsAll, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode ConflictResolutionMode
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

// TestConflictResolutionModeSupported tests that ConflictResolutionMode support
// detection works as expected.
func TestConflictResolutionModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode            ConflictResolutionMode
		ExpectSupported bool
	}{
		{ConflictResolutionMode_ConflictResolutionModeDefault, false},
		{ConflictResolutionMode_ConflictResolutionModeSafe, true},
		{ConflictResolutionMode_ConflictResolutionModeAlphaWins, true},
		{ConflictResolutionMode_ConflictResolutionModeBetaWins, true},
		{ConflictResolutionMode_ConflictResolutionModeAlphaWinsAll, true},
		{ConflictResolutionMode_ConflictResolutionModeBetaWinsAll, true},
		{(ConflictResolutionMode_ConflictResolutionModeBetaWinsAll + 1), false},
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

// TestConflictResolutionModeDescription tests that ConflictResolutionMode
// description generation works as expected.
func TestConflictResolutionModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode                ConflictResolutionMode
		ExpectedDescription string
	}{
		{ConflictResolutionMode_ConflictResolutionModeDefault, "Default"},
		{ConflictResolutionMode_ConflictResolutionModeSafe, "Safe"},
		{ConflictResolutionMode_ConflictResolutionModeAlphaWins, "Alpha Wins"},
		{ConflictResolutionMode_ConflictResolutionModeBetaWins, "Beta Wins"},
		{ConflictResolutionMode_ConflictResolutionModeAlphaWinsAll, "Alpha Wins All"},
		{ConflictResolutionMode_ConflictResolutionModeBetaWinsAll, "Beta Wins All"},
		{(ConflictResolutionMode_ConflictResolutionModeBetaWinsAll + 1), "Unknown"},
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
