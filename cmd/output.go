package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// StatusLinePrinter provides printing facilities for dynamically updating
// status lines in the console. It supports colorized printing.
type StatusLinePrinter struct {
	// UseStandardError causes the printer to use standard error for its output
	// instead of standard output (the default).
	UseStandardError bool
	// nonEmpty indicates whether or not the printer has printed any non-empty
	// content to the status line.
	nonEmpty bool
}

// Print prints a message to the status line, overwriting any existing content.
// Color escape sequences are supported. Messages will be truncated to a
// platform-dependent maximum length and padded appropriately.
func (p *StatusLinePrinter) Print(message string) {
	// Determine output stream.
	output := color.Output
	if p.UseStandardError {
		output = color.Error
	}

	// Print the message, prefixed with a carriage return to wipe out the
	// previous line (if any). Ensure that the status prints as a specified
	// width, truncating or right-padding with space as necessary. On POSIX
	// systems, this width is 80 characters and on Windows it's 79. The reason
	// for 79 on Windows is that for cmd.exe consoles the line width needs to be
	// narrower than the console (which is 80 columns by default) for carriage
	// return wipes to work (if it's the same width, the next carriage return
	// overflows to the next line, behaving exactly like a newline). We print to
	// the color output so that color escape sequences are properly handled - in
	// all other cases this will behave just like standard output.
	// TODO: We should probably try to detect the console width.
	fmt.Fprintf(output, statusLineFormat, message)

	// Update our non-empty status. We're always non-empty after printing
	// because we print padding as well.
	p.nonEmpty = true
}

// Clear clears any content on the status line and moves the cursor back to the
// beginning of the line.
func (p *StatusLinePrinter) Clear() {
	// Write over any existing data.
	p.Print("")

	// Determine output stream.
	output := os.Stdout
	if p.UseStandardError {
		output = os.Stderr
	}

	// Wipe out any existing line.
	fmt.Fprint(output, "\r")

	// Update our non-empty status.
	p.nonEmpty = false
}

// BreakIfNonEmpty prints a newline character if the current line is non-empty.
func (p *StatusLinePrinter) BreakIfNonEmpty() {
	// If the status line contents are non-empty, then print a newline and mark
	// ourselves as empty.
	if p.nonEmpty {
		// Determine output stream.
		output := os.Stdout
		if p.UseStandardError {
			output = os.Stderr
		}

		// Print a line break.
		fmt.Fprintln(output)

		// Update our non-empty status.
		p.nonEmpty = false
	}
}
