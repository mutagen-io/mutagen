package encoding

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
)

// testMessageJSON is a test structure to use for encoding tests using JSON.
type testMessageJSON struct {
	// Name represents a person's name.
	Name string
	// Age represents a person's age.
	Age uint
}

const (
	// testMessageJSONString is the JSON-encoded form of the JSON test data.
	testMessageJSONString = `{"Name":"George","Age":67}`
	// testMessageJSONName is the JSON test name.
	testMessageJSONName = "George"
	// testMessageJSONAge is the JSON test age.
	testMessageJSONAge = 67
)

// TestLoadAndUnmarshalNonExistentPath tests that loading fails from a
// non-existent path.
func TestLoadAndUnmarshalNonExistentPath(t *testing.T) {
	if !os.IsNotExist(LoadAndUnmarshal("/this/does/not/exist", nil)) {
		t.Error("expected LoadAndUnmarshal to pass through non-existence errors")
	}
}

// TestLoadAndUnmarshalDirectory tests that loading fails from a directory.
func TestLoadAndUnmarshalDirectory(t *testing.T) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Perform the test.
	if LoadAndUnmarshal(homeDirectory, nil) == nil {
		t.Error("expected LoadAndUnmarshal error when loading directory")
	}
}

// TestLoadAndUnmarshalUnmarshalFail tests that unmarshaling fails if the
// unmarshaling callback fails.
func TestLoadAndUnmarshalUnmarshalFail(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Create an empty temporary file and defer its cleanup.
	file, err := os.CreateTemp("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer must.OSRemove(file.Name(), logger)

	// Create a broken unmarshaling function.
	unmarshal := func(_ []byte) error {
		return errors.New("unmarshal failed")
	}

	// Attempt to load and unmarshal using a broken unmarshaling function.
	if LoadAndUnmarshal(file.Name(), unmarshal) == nil {
		t.Error("expected LoadAndUnmarshal to return an error")
	}
}

// TestLoadAndUnmarshal tests that loading and unmarshaling succeed.
func TestLoadAndUnmarshal(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Write the test JSON to a temporary file and defer its cleanup.
	file, err := os.CreateTemp("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if _, err = file.Write([]byte(testMessageJSONString)); err != nil {
		t.Fatal("unable to write data to temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer must.OSRemove(file.Name(), logger)

	// Create an unmarshaling function.
	value := &testMessageJSON{}
	unmarshal := func(data []byte) error {
		return json.Unmarshal(data, value)
	}

	// Attempt to load and unmarshal.
	if err := LoadAndUnmarshal(file.Name(), unmarshal); err != nil {
		t.Fatal("LoadAndUnmarshal failed:", err)
	}

	// Verify test value names.
	if value.Name != testMessageJSONName {
		t.Error("test message name mismatch:", value.Name, "!=", testMessageJSONName)
	}
	if value.Age != testMessageJSONAge {
		t.Error("test message age mismatch:", value.Age, "!=", testMessageJSONAge)
	}
}

// TestMarshalAndSaveMarshalFail tests that marshaling fails if the marshaling
// callback fails.
func TestMarshalAndSaveMarshalFail(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Create an empty temporary file and defer its cleanup.
	file, err := os.CreateTemp("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer must.OSRemove(file.Name(), logger)

	// Create a broken marshaling function.
	marshal := func() ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	// Attempt to marshal and save using a broken unmarshaling function.
	if MarshalAndSave(file.Name(), logger, marshal) == nil {
		t.Error("expected MarshalAndSave to return an error")
	}
}

// TestMarshalAndSaveOverDirectory tests that saving over a directory fails.
func TestMarshalAndSaveOverDirectory(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Create a marshaling function.
	marshal := func() ([]byte, error) {
		return []byte{0}, nil
	}

	// Attempt to marshal and save over a directory.
	if MarshalAndSave(t.TempDir(), logger, marshal) == nil {
		t.Error("expected MarshalAndSave to return an error")
	}
}

// TestMarshalAndSave tests that marshaling and saving succeed.
func TestMarshalAndSave(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Create an empty temporary file and defer its cleanup.
	file, err := os.CreateTemp("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer must.OSRemove(file.Name(), logger)

	// Create a marshaling function.
	value := &testMessageJSON{Name: testMessageJSONName, Age: testMessageJSONAge}
	marshal := func() ([]byte, error) {
		return json.Marshal(value)
	}

	// Attempt to marshal and save.
	if err := MarshalAndSave(file.Name(), logger, marshal); err != nil {
		t.Fatal("MarshalAndSave failed:", err)
	}

	// Read the contents of the file and ensure they match what's expected.
	// TODO: Are we relying too much on the implementation details of the JSON
	// encoder here?
	contents, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal("unable to read saved contents:", err)
	} else if string(contents) != testMessageJSONString {
		t.Error("marshaled contents do not match expected:", string(contents), "!=", testMessageJSONString)
	}
}
