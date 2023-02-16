package hashing

import (
	"testing"
)

// TestAlgorithmUnmarshal tests that unmarshaling from a string specification
// succeeeds for Algorithm.
func TestAlgorithmUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		text          string
		expected      Algorithm
		expectFailure bool
	}{
		{"", Algorithm_AlgorithmDefault, true},
		{"asdf", Algorithm_AlgorithmDefault, true},
		{"sha1", Algorithm_AlgorithmSHA1, false},
		{"sha256", Algorithm_AlgorithmSHA256, false},
		{"xxh128", Algorithm_AlgorithmXXH128, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var algorithm Algorithm
		if err := algorithm.UnmarshalText([]byte(testCase.text)); err != nil {
			if !testCase.expectFailure {
				t.Errorf("unable to unmarshal text (%s): %s", testCase.text, err)
			}
		} else if testCase.expectFailure {
			t.Error("unmarshaling succeeded unexpectedly for text:", testCase.text)
		} else if algorithm != testCase.expected {
			t.Errorf(
				"unmarshaled algorithm (%s) does not match expected (%s)",
				algorithm,
				testCase.expected,
			)
		}
	}
}

// TestAlgorithmSupportStatus tests that Algorithm support detection works as
// expected.
func TestAlgorithmSupportStatus(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		algorithm Algorithm
		expected  AlgorithmSupportStatus
	}{
		{Algorithm_AlgorithmDefault, AlgorithmSupportStatusUnsupported},
		{Algorithm_AlgorithmSHA1, AlgorithmSupportStatusSupported},
		{Algorithm_AlgorithmSHA256, AlgorithmSupportStatusSupported},
		{Algorithm_AlgorithmXXH128, xxh128SupportStatus()},
		{(Algorithm_AlgorithmXXH128 + 1), AlgorithmSupportStatusUnsupported},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if supportStatus := testCase.algorithm.SupportStatus(); supportStatus != testCase.expected {
			t.Errorf(
				"algorithm support status (%d) does not match expected (%d)",
				supportStatus,
				testCase.expected,
			)
		}
	}
}

// TestAlgorithmDescription tests that Algorithm description generation works as
// expected.
func TestAlgorithmDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		algorithm Algorithm
		expected  string
	}{
		{Algorithm_AlgorithmDefault, "Default"},
		{Algorithm_AlgorithmSHA1, "SHA-1"},
		{Algorithm_AlgorithmSHA256, "SHA-256"},
		{Algorithm_AlgorithmXXH128, "XXH128"},
		{(Algorithm_AlgorithmXXH128 + 1), "Unknown"},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if description := testCase.algorithm.Description(); description != testCase.expected {
			t.Errorf(
				"algorithm description (%s) does not match expected (%s)",
				description,
				testCase.expected,
			)
		}
	}
}
