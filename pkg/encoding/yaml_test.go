package encoding

import (
	"os"
	"testing"
)

// testMessageYAML is a test structure to use for encoding tests using YAML.
type testMessageYAML struct {
	Section struct {
		Name string `yaml:"name"`
		Age  uint   `yaml:"age"`
	} `yaml:"section"`
}

const (
	// testMessageYAMLString is the YAML-encoded form of the YAML test data.
	testMessageYAMLString = `
section:
  name: "Abraham"
  age: 56
`
	// testMessageYAMLName is the YAML test name.
	testMessageYAMLName = "Abraham"
	// testMessageYAMLAge is the YAML test age.
	testMessageYAMLAge = 56
)

// TestLoadAndUnmarshalYAML tests that loading and unmarshaling YAML data
// succeeds.
func TestLoadAndUnmarshalYAML(t *testing.T) {
	// Write the test YAML to a temporary file and defer its cleanup.
	file, err := os.CreateTemp("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if _, err = file.Write([]byte(testMessageYAMLString)); err != nil {
		t.Fatal("unable to write data to temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Attempt to load and unmarshal.
	value := &testMessageYAML{}
	if err := LoadAndUnmarshalYAML(file.Name(), value); err != nil {
		t.Fatal("loadAndUnmarshal failed:", err)
	}

	// Verify test value names.
	if value.Section.Name != testMessageYAMLName {
		t.Error("test message name mismatch:", value.Section.Name, "!=", testMessageYAMLName)
	}
	if value.Section.Age != testMessageYAMLAge {
		t.Error("test message age mismatch:", value.Section.Age, "!=", testMessageYAMLAge)
	}
}
