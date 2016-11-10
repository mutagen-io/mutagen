package ssh

import (
	"github.com/havoc-io/mutagen/process"
)

const (
	errorCodeCommandNotFound = 127
)

func IsCommandNotFound(err error) bool {
	// TODO: Figure out how to identify "command not found" errors for Windows
	// SSH servers.
	code, codeErr := process.ExitCodeForError(err)
	return codeErr == nil && code == errorCodeCommandNotFound
}
