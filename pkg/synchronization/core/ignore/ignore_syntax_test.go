package ignore

import (
	"testing"
)

// TestIgnoreSyntaxIsDefault tests IgnoreSyntax.IsDefault.
func TestIgnoreSyntaxIsDefault(t *testing.T) {
	// Define test cases.
	tests := []struct {
		value    IgnoreSyntax
		expected bool
	}{
		{IgnoreSyntax_IgnoreSyntaxDefault - 1, false},
		{IgnoreSyntax_IgnoreSyntaxDefault, true},
		{IgnoreSyntax_IgnoreSyntaxGit, false},
		{IgnoreSyntax_IgnoreSyntaxDocker, false},
		{IgnoreSyntax_IgnoreSyntaxDocker + 1, false},
	}

	// Process test cases.
	for i, test := range tests {
		if result := test.value.IsDefault(); result && !test.expected {
			t.Errorf("test index %d: value was unexpectedly classified as default", i)
		} else if !result && test.expected {
			t.Errorf("test index %d: value was unexpectedly classified as non-default", i)
		}
	}
}

// TestIgnoreSyntaxUnmarshalText tests IgnoreSyntax.UnmarshalText.
func TestIgnoreSyntaxUnmarshalText(t *testing.T) {
	// Define test cases.
	tests := []struct {
		text          string
		expectedMode  IgnoreSyntax
		expectFailure bool
	}{
		{"", IgnoreSyntax_IgnoreSyntaxDefault, true},
		{"asdf", IgnoreSyntax_IgnoreSyntaxDefault, true},
		{"git", IgnoreSyntax_IgnoreSyntaxGit, false},
		{"docker", IgnoreSyntax_IgnoreSyntaxDocker, false},
	}

	// Process test cases.
	for _, test := range tests {
		var mode IgnoreSyntax
		if err := mode.UnmarshalText([]byte(test.text)); err != nil {
			if !test.expectFailure {
				t.Errorf("unable to unmarshal text (%s): %s", test.text, err)
			}
		} else if test.expectFailure {
			t.Error("unmarshaling succeeded unexpectedly for text:", test.text)
		} else if mode != test.expectedMode {
			t.Errorf(
				"unmarshaled mode (%s) does not match expected (%s)",
				mode,
				test.expectedMode,
			)
		}
	}
}

// TestIgnoreSyntaxSupported tests IgnoreSyntax.Supported.
func TestIgnoreSyntaxSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode            IgnoreSyntax
		expectSupported bool
	}{
		{IgnoreSyntax_IgnoreSyntaxDefault, false},
		{IgnoreSyntax_IgnoreSyntaxGit, true},
		{IgnoreSyntax_IgnoreSyntaxDocker, true},
		{(IgnoreSyntax_IgnoreSyntaxDocker + 1), false},
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

// TestIgnoreSyntaxDescription tests IgnoreSyntax.Description.
func TestIgnoreSyntaxDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode                IgnoreSyntax
		expectedDescription string
	}{
		{IgnoreSyntax_IgnoreSyntaxDefault, "Default"},
		{IgnoreSyntax_IgnoreSyntaxGit, "Git"},
		{IgnoreSyntax_IgnoreSyntaxDocker, "Docker"},
		{(IgnoreSyntax_IgnoreSyntaxDocker + 1), "Unknown"},
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
