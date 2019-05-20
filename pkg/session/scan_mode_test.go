package session

import (
	"testing"
)

// TestScanModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for ScanMode.
func TestScanModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		text          string
		expectedMode  ScanMode
		expectFailure bool
	}{
		{"", ScanMode_ScanModeDefault, true},
		{"asdf", ScanMode_ScanModeDefault, true},
		{"full", ScanMode_ScanModeFull, false},
		{"accelerated", ScanMode_ScanModeAccelerated, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode ScanMode
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

// TestScanModeSupported tests that ScanMode support detection works as
// expected.
func TestScanModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode            ScanMode
		expectSupported bool
	}{
		{ScanMode_ScanModeDefault, false},
		{ScanMode_ScanModeFull, true},
		{ScanMode_ScanModeAccelerated, true},
		{(ScanMode_ScanModeAccelerated + 1), false},
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

// TestScanModeDescription tests that ScanMode description generation works as
// expected.
func TestScanModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode                ScanMode
		expectedDescription string
	}{
		{ScanMode_ScanModeDefault, "Default"},
		{ScanMode_ScanModeFull, "Full"},
		{ScanMode_ScanModeAccelerated, "Accelerated"},
		{(ScanMode_ScanModeAccelerated + 1), "Unknown"},
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
