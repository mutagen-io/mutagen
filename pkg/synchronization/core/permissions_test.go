package core

import (
	"testing"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// TestAnyExecutableBitSet tests anyExecutableBitSet.
func TestAnyExecutableBitSet(t *testing.T) {
	// Define test cases.
	tests := []struct {
		value    filesystem.Mode
		expected bool
	}{
		{0111, true},
		{0222, false},
		{0444, false},
		{0555, true},
		{0666, false},
		{0766, true},
		{0676, true},
		{0667, true},
		{0776, true},
		{0677, true},
		{0767, true},
		{0777, true},
	}

	// Process test cases.
	for i, test := range tests {
		if result := anyExecutableBitSet(test.value); result && !test.expected {
			t.Errorf("test index %d: mode unexpectedly has executability bit(s) set", i)
		} else if !result && test.expected {
			t.Errorf("test index %d: mode unexpectedly has no executability bits set", i)
		}
	}
}

// TestEnsureDefaultFileModeValid tests EnsureDefaultFileModeValid.
func TestEnsureDefaultFileModeValid(t *testing.T) {
	// Define test cases.
	tests := []struct {
		value    filesystem.Mode
		expected bool
	}{
		{0, false},
		{01000, false},
		{0100, false},
		{0010, false},
		{0001, false},
		{0111, false},
		{0222, true},
		{0444, true},
		{0666, true},
		{0644, true},
		{0777, false},
		{0766, false},
		{0676, false},
		{0667, false},
	}

	// Process test cases.
	for i, test := range tests {
		if err := EnsureDefaultFileModeValid(test.value); err == nil && !test.expected {
			t.Errorf("test index %d: mode unexpectedly classified as valid", i)
		} else if err != nil && test.expected {
			t.Errorf("test index %d: mode unexpectedly classified as invalid: %v", i, err)
		}
	}
}

// TestEnsureDefaultDirectoryModeValid tests EnsureDefaultDirectoryModeValid.
func TestEnsureDefaultDirectoryModeValid(t *testing.T) {
	// Define test cases.
	tests := []struct {
		value    filesystem.Mode
		expected bool
	}{
		{0, false},
		{01000, false},
		{0100, true},
		{0010, true},
		{0001, true},
		{0111, true},
		{0222, true},
		{0444, true},
		{0666, true},
		{0777, true},
		{0766, true},
		{0676, true},
		{0667, true},
	}

	// Process test cases.
	for i, test := range tests {
		if err := EnsureDefaultDirectoryModeValid(test.value); err == nil && !test.expected {
			t.Errorf("test index %d: mode unexpectedly classified as valid", i)
		} else if err != nil && test.expected {
			t.Errorf("test index %d: mode unexpectedly classified as invalid: %v", i, err)
		}
	}
}

// TestMarkExecutableForReaders tests markExecutableForReaders.
func TestMarkExecutableForReaders(t *testing.T) {
	// Define test cases.
	tests := []struct {
		value    filesystem.Mode
		expected filesystem.Mode
	}{
		{0111, 0111},
		{0222, 0222},
		{0622, 0722},
		{0262, 0272},
		{0226, 0227},
		{0662, 0772},
		{0266, 0277},
		{0626, 0727},
		{0722, 0722},
		{0272, 0272},
		{0227, 0227},
		{0772, 0772},
		{0277, 0277},
		{0727, 0727},
	}

	// Process test cases.
	for i, test := range tests {
		if result := markExecutableForReaders(test.value); result != test.expected {
			t.Errorf("test index %d: result does not match expected: %#o != %#o",
				i, result, test.expected,
			)
		}
	}
}
