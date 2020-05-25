package environment

import (
	"testing"
)

// TestParseBlock tests ParseBlock.
func TestParseBlock(t *testing.T) {
	// Set test parameters.
	input := "KEY=VALUE\nKEY=duplicate\r\nOTHER=2\nIGNORED\n\n"
	expected := []string{
		"KEY=VALUE",
		"KEY=duplicate",
		"OTHER=2",
		"IGNORED",
	}

	// Perform parsing.
	output := ParseBlock(input)

	// Validate results.
	if len(output) != len(expected) {
		t.Fatal("output length does not match expected:", len(output), "!=", len(expected))
	}
	for v, value := range output {
		if value != expected[v] {
			t.Error("output value at index", v, "does not match expected:", value, "!=", expected[v])
		}
	}
}
