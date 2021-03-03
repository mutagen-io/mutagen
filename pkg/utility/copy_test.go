package utility

import (
	"testing"
)

// TestCopyStringSlice tests CopyStringSlice.
func TestCopyStringSlice(t *testing.T) {
	// Set up test cases.
	testCases := [][]string{
		nil,
		{},
		{"cat"},
		{"cat", "dog"},
	}

	// Process test cases.
	for _, value := range testCases {
		if result := CopyStringSlice(value); value == nil && result != nil {
			t.Error("nilness not preserved by copy")
		} else if !StringSlicesEqual(result, value) {
			t.Error("copy result not equal to original")
		}
	}
}

// TestCopyStringMap tests CopyStringMap.
func TestCopyStringMap(t *testing.T) {
	// Set up test cases.
	testCases := []map[string]string{
		nil,
		{},
		{"cat": "meow"},
		{"cat": "meow", "dog": "bark"},
	}

	// Process test cases.
	for _, value := range testCases {
		if result := CopyStringMap(value); value == nil && result != nil {
			t.Error("nilness not preserved by copy")
		} else if !StringMapsEqual(result, value) {
			t.Error("copy result not equal to original")
		}
	}
}
