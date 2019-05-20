package behavior

import (
	"testing"
)

// TestProbeModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for ProbeMode.
func TestProbeModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		text          string
		expectedMode  ProbeMode
		expectFailure bool
	}{
		{"", ProbeMode_ProbeModeDefault, true},
		{"asdf", ProbeMode_ProbeModeDefault, true},
		{"probe", ProbeMode_ProbeModeProbe, false},
		{"assume", ProbeMode_ProbeModeAssume, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode ProbeMode
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

// TestProbeModeSupported tests that ProbeMode support detection works as
// expected.
func TestProbeModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode            ProbeMode
		expectSupported bool
	}{
		{ProbeMode_ProbeModeDefault, false},
		{ProbeMode_ProbeModeProbe, true},
		{ProbeMode_ProbeModeAssume, true},
		{(ProbeMode_ProbeModeAssume + 1), false},
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

// TestProbeModeDescription tests that ProbeMode description generation works as
// expected.
func TestProbeModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode                ProbeMode
		expectedDescription string
	}{
		{ProbeMode_ProbeModeDefault, "Default"},
		{ProbeMode_ProbeModeProbe, "Probe"},
		{ProbeMode_ProbeModeAssume, "Assume"},
		{(ProbeMode_ProbeModeAssume + 1), "Unknown"},
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
