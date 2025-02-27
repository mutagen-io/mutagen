package project

import "os"

const (
	// DefaultConfigurationFileName is the name of the Mutagen project
	// configuration file.
	DefaultConfigurationFileName = "mutagen.yml"
	// LockFileExtension is the extension added to a configuration file path in
	// order to compute the corresponding lock file.
	LockFileExtension = ".lock"
)

func ConfigurationFileName() string {
  fileName := os.Getenv("MUTAGEN_PROJECT_FILE")
  if fileName == "" {
    return DefaultConfigurationFileName
  }
  return fileName
}
