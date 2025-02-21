//go:build:windows
package must

import (
	"github.com/mutagen-io/mutagen/pkg/logging"
	"golang.org/x/sys/windows"
)

func CloseWindowsHandle(wh windows.Handle, logger *logging.Logger) {
	err := windows.CloseHandle(wh)
	if err != nil {
		logger.Warnf("Unable to close handle %d: %s", wh, err.Error())
	}
}
