package encoding

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// testMessageJSON is a test structure to use for encoding tests using JSON.
type testMessageJSON struct {
	// Name represents a person's name.
	Name string
	// Age represent's a person's age.
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
	if !os.IsNotExist(loadAndUnmarshal("/this/does/not/exist", nil)) {
		t.Error("expected loadAndUnmarshal to pass through non-existence errors")
	}
}

// TestLoadAndUnmarshalDirectory tests that loading fails from a directory.
func TestLoadAndUnmarshalDirectory(t *testing.T) {
	if loadAndUnmarshal(filesystem.HomeDirectory, nil) == nil {
		t.Error("expected loadAndUnmarshal error when loading directory")
	}
}

// TestLoadAndUnmarshalUnmarshalFail tests that unmarshaling fails if the
// unmarshaling callback fails.
func TestLoadAndUnmarshalUnmarshalFail(t *testing.T) {
	// Create an empty temporary file and defer its cleanup.
	file, err := ioutil.TempFile("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Create a broken unmarshaling function.
	unmarshal := func(_ []byte) error {
		return errors.New("unmarshal failed")
	}

	// Attempt to load and unmarshal using a broken unmarshaling function.
	if loadAndUnmarshal(file.Name(), unmarshal) == nil {
		t.Error("expected loadAndUnmarshal to return an error")
	}
}

// TestLoadAndUnmarshal tests that loading and unmarshaling succeed.
func TestLoadAndUnmarshal(t *testing.T) {
	// Write the test JSON to a temporary file and defer its cleanup.
	file, err := ioutil.TempFile("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if _, err = file.Write([]byte(testMessageJSONString)); err != nil {
		t.Fatal("unable to write data to temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Create an unmarshaling function.
	value := &testMessageJSON{}
	unmarshal := func(data []byte) error {
		return json.Unmarshal(data, value)
	}

	// Attempt to load and unmarshal.
	if err := loadAndUnmarshal(file.Name(), unmarshal); err != nil {
		t.Fatal("loadAndUnmarshal failed:", err)
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
	// Create an empty temporary file and defer its cleanup.
	file, err := ioutil.TempFile("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Create a broken marshaling function.
	marshal := func() ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	// Attempt to marshal and save using a broken unmarshaling function.
	if marshalAndSave(file.Name(), marshal) == nil {
		t.Error("expected marshalAndSave to return an error")
	}
}

// TestMarshalAndSaveInvalidPath tests that saving to an invalid path fails.
func TestMarshalAndSaveInvalidPath(t *testing.T) {
	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Create a marshaling function.
	marshal := func() ([]byte, error) {
		return []byte{0}, nil
	}

	// Attempt to marshal and save using an invalid path.
	if marshalAndSave(directory, marshal) == nil {
		t.Error("expected marshalAndSave to return an error")
	}
}

// TestMarshalAndSave tests that marshaling and saving succeed.
func TestMarshalAndSave(t *testing.T) {
	// Create an empty temporary file and defer its cleanup.
	file, err := ioutil.TempFile("", "mutagen_encoding")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Create a marshaling function.
	value := &testMessageJSON{Name: testMessageJSONName, Age: testMessageJSONAge}
	marshal := func() ([]byte, error) {
		return json.Marshal(value)
	}

	// Attempt to marshal and save.
	if err := marshalAndSave(file.Name(), marshal); err != nil {
		t.Fatal("marshalAndSave failed:", err)
	}

	// Read the contents of the file and ensure they match what's expected.
	// TODO: Are we relying too much on the implementation details of the JSON
	// encoder here?
	contents, err := ioutil.ReadFile(file.Name())
	if err != nil {
		t.Fatal("unable to read saved contents:", err)
	} else if string(contents) != testMessageJSONString {
		t.Error("marshaled contents do not match expected:", string(contents), "!=", testMessageJSONString)
	}
}
