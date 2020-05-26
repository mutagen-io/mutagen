package environment

import (
	"testing"
)

// TestToMap tests ToMap.
func TestToMap(t *testing.T) {
	// Set test parameters.
	input := []string{
		"KEY=VALUE",
		"KEY=duplicate",
		"OTHER=2",
		"IGNORED",
	}
	expected := map[string]string{
		"KEY":   "duplicate",
		"OTHER": "2",
	}

	// Perform conversion.
	output := ToMap(input)

	// Validate results.
	if len(output) != len(expected) {
		t.Fatal("output length does not match expected:", len(output), "!=", len(expected))
	}
	for key, value := range output {
		if expectedValue, ok := expected[key]; !ok {
			t.Errorf("output key \"%s\" not expected", key)
		} else if value != expectedValue {
			t.Error("output value does not match expected:", value, "!=", expectedValue)
		}
	}
}

// TestFromMap tests FromMap.
func TestFromMap(t *testing.T) {
	// Set test parameters.
	input := map[string]string{
		"KEY":   "duplicate",
		"OTHER": "2",
		"HEY":   "THERE",
	}

	// Perform conversion to a slice and then back to a map so that we can
	// compare based on map contents.
	output := ToMap(FromMap(input))

	// Validate results.
	if len(output) != len(input) {
		t.Fatal("output length does not match expected:", len(output), "!=", len(input))
	}
	for key, value := range output {
		if expectedValue, ok := input[key]; !ok {
			t.Errorf("output key \"%s\" not expected", key)
		} else if value != expectedValue {
			t.Error("output value does not match expected:", value, "!=", expectedValue)
		}
	}
}

// TestFromMapNilMap tests FromMap with a nil input.
func TestFromMapNilMap(t *testing.T) {
	if FromMap(nil) != nil {
		t.Fatal("FromMap returned non-nil result with nil input")
	}
}

// TestFromMapEmptyMap tests FromMap with an empty input map.
func TestFromMapEmptyMap(t *testing.T) {
	result := FromMap(make(map[string]string))
	if result == nil {
		t.Fatal("FromMap returned nil result with non-nil input")
	} else if len(result) != 0 {
		t.Fatal("FromMap returned non-empty result with empty input")
	}
}
