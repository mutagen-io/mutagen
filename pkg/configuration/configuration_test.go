package configuration

import (
	"io/ioutil"
	"os"
	"testing"
)

const (
	testConfigurationGibberish = "[a+1a4"
	testConfigurationValid     = `[sync]
mode = "two-way-resolved"
maxEntryCount = 500
maxStagingFileSize = "1000 GB"

[symlink]
mode = "portable"

[watch]
mode = "force-poll"
pollingInterval = 5

[ignore]
default = ["ignore/this/**", "!ignore/this/that"]

[permissions]
defaultFileMode = 644
defaultDirectoryMode = 0755
defaultOwner = "george"
defaultGroup = "presidents"
`
)

func TestLoadNonExistent(t *testing.T) {
	if c, err := loadFromPath("/this/does/not/exist"); err != nil {
		t.Error("load from non-existent path failed:", err)
	} else if c == nil {
		t.Error("load from non-existent path returned nil configuration")
	}
}

func TestLoadEmpty(t *testing.T) {
	// Create an empty temporary file and defer its cleanup.
	file, err := ioutil.TempFile("", "mutagen_configuration")
	if err != nil {
		t.Error("unable to create temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Error("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Attempt to load.
	if c, err := loadFromPath(file.Name()); err != nil {
		t.Error("load from empty file failed:", err)
	} else if c == nil {
		t.Error("load from empty file returned nil configuration")
	}
}

func TestLoadGibberish(t *testing.T) {
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
	if _, err := loadFromPath(file.Name()); err == nil {
		t.Error("load did not fail on gibberish configuration")
	}
}

func TestLoadDirectory(t *testing.T) {
	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_configuration")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Attempt to load.
	if _, err := loadFromPath(directory); err == nil {
		t.Error("load did not fail on directory path")
	}
}

func TestLoadValidConfiguration(t *testing.T) {
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
	if c, err := loadFromPath(file.Name()); err != nil {
		t.Error("load from valid configuration failed:", err)
	} else if c == nil {
		t.Error("load from valid configuration returned nil configuration")
	}
}

// NOTE: This test depends on not having an invalid ~/.mutagen.toml file.
func TestLoad(t *testing.T) {
	if c, err := Load(); err != nil {
		t.Error("load failed:", err)
	} else if c == nil {
		t.Error("load returned nil configuration")
	}
}
