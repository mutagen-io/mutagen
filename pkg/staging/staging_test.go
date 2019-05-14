package staging

import (
	"testing"
)

// TestStagingModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for StagingMode.
func TestStagingModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Text          string
		ExpectedMode  StagingMode
		ExpectFailure bool
	}{
		{"", StagingMode_StagingModeDefault, true},
		{"asdf", StagingMode_StagingModeDefault, true},
		{"mutagen", StagingMode_StagingModeMutagen, false},
		{"neighboring", StagingMode_StagingModeNeighboring, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode StagingMode
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

// TestStagingModeSupported tests that StagingMode support detection works as
// expected.
func TestStagingModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode            StagingMode
		ExpectSupported bool
	}{
		{StagingMode_StagingModeDefault, false},
		{StagingMode_StagingModeMutagen, true},
		{StagingMode_StagingModeNeighboring, true},
		{(StagingMode_StagingModeNeighboring + 1), false},
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

// TestStagingModeDescription tests that StagingMode description generation
// works as expected.
func TestStagingModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode                StagingMode
		ExpectedDescription string
	}{
		{StagingMode_StagingModeDefault, "Default"},
		{StagingMode_StagingModeMutagen, "Mutagen"},
		{StagingMode_StagingModeNeighboring, "Neighboring"},
		{(StagingMode_StagingModeNeighboring + 1), "Unknown"},
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
