package encoding

import (
	"io/ioutil"
	"os"
	"testing"
)

// testMessageTOML is a test structure to use for encoding tests using TOML.
type testMessageTOML struct {
	Section struct {
		Name string `toml:"name"`
		Age  uint   `toml:"age"`
	} `toml:"section"`
}

const (
	// testMessageTOMLString is the TOML-encoded form of the TOML test data.
	testMessageTOMLString = `
[section]
name= "Abraham"
age=56
`
	// testMessageTOMLName is the TOML test name.
	testMessageTOMLName = "Abraham"
	// testMessageTOMLAge is the TOML test age.
	testMessageTOMLAge = 56
)

// TestLoadAndUnmarshalTOML tests that loading and unmarshaling TOML data
// succeeds.
func TestLoadAndUnmarshalTOML(t *testing.T) {
	// Write the test TOML to a temporary file and defer its cleanup.
	file, err := ioutil.TempFile("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if _, err = file.Write([]byte(testMessageTOMLString)); err != nil {
		t.Fatal("unable to write data to temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Attempt to load and unmarshal.
	value := &testMessageTOML{}
	if err := LoadAndUnmarshalTOML(file.Name(), value); err != nil {
		t.Fatal("loadAndUnmarshal failed:", err)
	}

	// Verify test value names.
	if value.Section.Name != testMessageTOMLName {
		t.Error("test message name mismatch:", value.Section.Name, "!=", testMessageTOMLName)
	}
	if value.Section.Age != testMessageTOMLAge {
		t.Error("test message age mismatch:", value.Section.Age, "!=", testMessageTOMLAge)
	}
}
