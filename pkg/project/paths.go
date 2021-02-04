package project

import (
	"crypto/sha1"
	"fmt"
	"path/filepath"
)

const (
	// DefaultConfigurationFileName is the name of the Mutagen project
	// configuration file.
	DefaultConfigurationFileName = "mutagen.yml"
	// LockFileExtension is the extension added to a configuration file path in
	// order to compute the corresponding lock file.
	LockFileExtension = ".lock"
)

func LockfilePath(configPath string) (string, error) {
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return "", err
	}

	absConfigPathHash := fmt.Sprintf("%x", sha1.Sum([]byte(absConfigPath)))
	return absConfigPathHash + LockFileExtension, nil
}
