package sync

import (
	"testing"
)

// TestPermissionExposureLevelUnmarshal tests that unmarshaling from a string
// specification succeeeds for PermissionExposureLevel.
func TestPermissionExposureLevelUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Text          string
		ExpectedMode  PermissionExposureLevel
		ExpectFailure bool
	}{
		{"", PermissionExposureLevel_PermissionExposureLevelDefault, true},
		{"asdf", PermissionExposureLevel_PermissionExposureLevelDefault, true},
		{"user", PermissionExposureLevel_PermissionExposureLevelUser, false},
		{"group", PermissionExposureLevel_PermissionExposureLevelGroup, false},
		{"other", PermissionExposureLevel_PermissionExposureLevelOther, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode PermissionExposureLevel
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

// TestPermissionExposureLevelSupported tests that PermissionExposureLevel
// support detection works as expected.
func TestPermissionExposureLevelSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode            PermissionExposureLevel
		ExpectSupported bool
	}{
		{PermissionExposureLevel_PermissionExposureLevelDefault, false},
		{PermissionExposureLevel_PermissionExposureLevelUser, true},
		{PermissionExposureLevel_PermissionExposureLevelGroup, true},
		{PermissionExposureLevel_PermissionExposureLevelOther, true},
		{(PermissionExposureLevel_PermissionExposureLevelOther + 1), false},
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

// TestPermissionExposureLevelDescription tests that PermissionExposureLevel
// description generation works as expected.
func TestPermissionExposureLevelDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode                PermissionExposureLevel
		ExpectedDescription string
	}{
		{PermissionExposureLevel_PermissionExposureLevelDefault, "Default"},
		{PermissionExposureLevel_PermissionExposureLevelUser, "User"},
		{PermissionExposureLevel_PermissionExposureLevelGroup, "Group"},
		{PermissionExposureLevel_PermissionExposureLevelOther, "Other"},
		{(PermissionExposureLevel_PermissionExposureLevelOther + 1), "Unknown"},
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

func TestAnyExecutableBitSet(t *testing.T) {
	if anyExecutableBitSet(0666) {
		t.Error("executable bits detected")
	}
	if !anyExecutableBitSet(0766) {
		t.Error("user executable bit not detected")
	}
	if !anyExecutableBitSet(0676) {
		t.Error("group executable bit not detected")
	}
	if !anyExecutableBitSet(0667) {
		t.Error("others executable bit not detected")
	}
	if !anyExecutableBitSet(0776) {
		t.Error("user executable bits not detected")
	}
	if !anyExecutableBitSet(0677) {
		t.Error("group executable bits not detected")
	}
	if !anyExecutableBitSet(0767) {
		t.Error("others executable bits not detected")
	}
	if !anyExecutableBitSet(0777) {
		t.Error("others executable bits not detected")
	}
}

func TestStripExecutableBits(t *testing.T) {
	if stripExecutableBits(0777) != 0666 {
		t.Error("executable bits not stripped")
	}
	if stripExecutableBits(0766) != 0666 {
		t.Error("user executable bit not stripped")
	}
	if stripExecutableBits(0676) != 0666 {
		t.Error("group executable bit not stripped")
	}
	if stripExecutableBits(0667) != 0666 {
		t.Error("others executable bit not stripped")
	}
}

func TestMarkExecutableForReaders(t *testing.T) {
	if markExecutableForReaders(0222) != 0222 {
		t.Error("erroneous executable bits added")
	}
	if markExecutableForReaders(0622) != 0722 {
		t.Error("incorrect executable bits added for user-readable file")
	}
	if markExecutableForReaders(0262) != 0272 {
		t.Error("incorrect executable bits added for group-readable file")
	}
	if markExecutableForReaders(0226) != 0227 {
		t.Error("incorrect executable bits added for others-readable file")
	}
}
