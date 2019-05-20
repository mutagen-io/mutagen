package configuration

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadSessionConfigurationNonExistent(t *testing.T) {
	if c, err := LoadSessionConfiguration("/this/does/not/exist"); err != nil {
		t.Error("load from non-existent path failed:", err)
	} else if c == nil {
		t.Error("load from non-existent path returned nil configuration")
	}
}

func TestLoadSessionConfigurationEmpty(t *testing.T) {
	// Create an empty temporary file and defer its cleanup.
	file, err := ioutil.TempFile("", "mutagen_configuration")
	if err != nil {
		t.Error("unable to create temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Error("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Attempt to load.
	if c, err := LoadSessionConfiguration(file.Name()); err != nil {
		t.Error("load from empty file failed:", err)
	} else if c == nil {
		t.Error("load from empty file returned nil configuration")
	}
}

func TestLoadSessionConfigurationGibberish(t *testing.T) {
	// Write gibberish to a temporary file and defer its cleanup.
	file, err := ioutil.TempFile("", "mutagen_configuration")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if _, err = file.Write([]byte(testConfigurationGibberish)); err != nil {
		t.Fatal("unable to write data to temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Attempt to load.
	if _, err := Load(file.Name()); err == nil {
		t.Error("load did not fail on gibberish configuration")
	}
}

func TestLoadSessionConfigurationDirectory(t *testing.T) {
	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_configuration")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Attempt to load.
	if _, err := Load(directory); err == nil {
		t.Error("load did not fail on directory path")
	}
}

func TestLoadSessionConfigurationValidConfiguration(t *testing.T) {
	// Write a valid configuration to a temporary file and defer its cleanup.
	file, err := ioutil.TempFile("", "mutagen_configuration")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if _, err = file.Write([]byte(testConfigurationValid)); err != nil {
		t.Fatal("unable to write data to temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Attempt to load.
	if c, err := Load(file.Name()); err != nil {
		t.Error("load from valid configuration failed:", err)
	} else if c == nil {
		t.Error("load from valid configuration returned nil configuration")
	}
}

// TODO: Add reflection-based test to ensure that all session configuration
// fields are populated.
