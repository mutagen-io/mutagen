package behavior

import (
	"testing"
)

// TestProbeModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for ProbeMode.
func TestProbeModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Text          string
		ExpectedMode  ProbeMode
		ExpectFailure bool
	}{
		{"", ProbeMode_ProbeModeDefault, true},
		{"asdf", ProbeMode_ProbeModeDefault, true},
		{"probe", ProbeMode_ProbeModeProbe, false},
		{"assume", ProbeMode_ProbeModeAssume, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode ProbeMode
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

// TestProbeModeSupported tests that ProbeMode support detection works as
// expected.
func TestProbeModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode            ProbeMode
		ExpectSupported bool
	}{
		{ProbeMode_ProbeModeDefault, false},
		{ProbeMode_ProbeModeProbe, true},
		{ProbeMode_ProbeModeAssume, true},
		{(ProbeMode_ProbeModeAssume + 1), false},
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

// TestProbeModeDescription tests that ProbeMode description generation works as
// expected.
func TestProbeModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode                ProbeMode
		ExpectedDescription string
	}{
		{ProbeMode_ProbeModeDefault, "Default"},
		{ProbeMode_ProbeModeProbe, "Probe"},
		{ProbeMode_ProbeModeAssume, "Assume"},
		{(ProbeMode_ProbeModeAssume + 1), "Unknown"},
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
