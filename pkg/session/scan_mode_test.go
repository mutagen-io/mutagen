package session

import (
	"testing"
)

// TestScanModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for ScanMode.
func TestScanModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Text          string
		ExpectedMode  ScanMode
		ExpectFailure bool
	}{
		{"", ScanMode_ScanModeDefault, true},
		{"asdf", ScanMode_ScanModeDefault, true},
		{"full", ScanMode_ScanModeFull, false},
		{"accelerated", ScanMode_ScanModeAccelerated, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode ScanMode
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

// TestScanModeSupported tests that ScanMode support detection works as
// expected.
func TestScanModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode            ScanMode
		ExpectSupported bool
	}{
		{ScanMode_ScanModeDefault, false},
		{ScanMode_ScanModeFull, true},
		{ScanMode_ScanModeAccelerated, true},
		{(ScanMode_ScanModeAccelerated + 1), false},
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

// TestScanModeDescription tests that ScanMode description generation works as
// expected.
func TestScanModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode                ScanMode
		ExpectedDescription string
	}{
		{ScanMode_ScanModeDefault, "Default"},
		{ScanMode_ScanModeFull, "Full"},
		{ScanMode_ScanModeAccelerated, "Accelerated"},
		{(ScanMode_ScanModeAccelerated + 1), "Unknown"},
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
