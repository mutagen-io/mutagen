package ssh

import (
	"github.com/havoc-io/mutagen/pkg/process"
)

const (
	errorCodeCommandNotFound = 127
)

func IsCommandNotFound(err error) bool {
	// TODO: Figure out how to identify "command not found" errors for Windows
	// SSH servers. POSIX shells generally return 127 in these cases, but I
	// don't know what Windows shells will do.
	code, codeErr := process.ExitCodeForError(err)
	return codeErr == nil && code == errorCodeCommandNotFound
}
