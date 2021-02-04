package project

import (
	"crypto/sha1"
	"fmt"
	"github.com/pkg/errors"
	"path/filepath"

	"os"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
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

	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	projectLockDir := filepath.Join(homeDirectory, filesystem.MutagenDataDirectoryName, filesystem.MutagenProjectLockDirectoryName)
	fileInfo, err := os.Stat(projectLockDir)

	if os.IsNotExist(err) {
		err := os.Mkdir(projectLockDir, 0700)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	} else if !fileInfo.IsDir() {
		return "", errors.New("project lock dir is not a directory")
	}

	absConfigPathHash := fmt.Sprintf("%x", sha1.Sum([]byte(absConfigPath)))
	return filepath.Join(homeDirectory, filesystem.MutagenDataDirectoryName, filesystem.MutagenProjectLockDirectoryName, absConfigPathHash + LockFileExtension), nil
}
