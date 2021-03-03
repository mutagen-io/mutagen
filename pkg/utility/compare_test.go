package utility

import (
	"testing"
)

// TestStringSlicesEqual tests StringSlicesEqual.
func TestStringSlicesEqual(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		first    []string
		second   []string
		expected bool
	}{
		{nil, nil, true},
		{[]string{}, nil, true},
		{nil, []string{}, true},
		{[]string{}, []string{}, true},
		{[]string{"cat"}, nil, false},
		{nil, []string{"dog"}, false},
		{[]string{"cat"}, []string{}, false},
		{[]string{}, []string{"dog"}, false},
		{[]string{"cat"}, []string{"dog"}, false},
		{[]string{"cat"}, []string{"cat", "dog"}, false},
		{[]string{"cat", "dog"}, []string{"cat"}, false},
		{[]string{"cat"}, []string{"cat"}, true},
		{[]string{"cat", "dog"}, []string{"cat", "dog"}, true},
		{[]string{"cat", "dog"}, []string{"dog", "cat"}, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if equal := StringSlicesEqual(testCase.first, testCase.second); equal != testCase.expected {
			t.Errorf("unexpected comparison result: %v == %v? %t (expected %t)",
				testCase.first, testCase.second,
				equal, testCase.expected,
			)
		}
	}
}

// TestStringMapsEqual tests StringMapsEqual.
func TestStringMapsEqual(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		first    map[string]string
		second   map[string]string
		expected bool
	}{
		{nil, nil, true},
		{map[string]string{}, nil, true},
		{nil, map[string]string{}, true},
		{map[string]string{}, map[string]string{}, true},
		{map[string]string{"cat": "meow"}, nil, false},
		{nil, map[string]string{"dog": "bark"}, false},
		{map[string]string{"cat": "meow"}, map[string]string{}, false},
		{map[string]string{}, map[string]string{"dog": "bark"}, false},
		{map[string]string{"cat": "meow"}, map[string]string{"dog": "bark"}, false},
		{map[string]string{"cat": "meow"}, map[string]string{"cat": "meow", "dog": "bark"}, false},
		{map[string]string{"cat": "meow", "dog": "bark"}, map[string]string{"cat": "meow"}, false},
		{map[string]string{"cat": "meow"}, map[string]string{"cat": "meow"}, true},
		{map[string]string{"cat": "meow", "dog": "bark"}, map[string]string{"cat": "meow", "dog": "bark"}, true},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if equal := StringMapsEqual(testCase.first, testCase.second); equal != testCase.expected {
			t.Errorf("unexpected comparison result: %v == %v? %t (expected %t)",
				testCase.first, testCase.second,
				equal, testCase.expected,
			)
		}
	}
}
