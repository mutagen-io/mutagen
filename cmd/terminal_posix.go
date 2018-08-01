// +build !windows

package cmd

// HandleTerminalCompatibility automatically restarts the current process inside
// a terminal compatibility emulator if necessary. It currently only handles the
// case of mintty consoles on Windows requiring a relaunch of the current
// command inside winpty.
func HandleTerminalCompatibility() {
	// No terminal emulation is required on POSIX systems.
}
