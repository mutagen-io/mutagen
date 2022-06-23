package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/pkg/prompting"
)

// StatusLinePrinter provides printing facilities for dynamically updating
// status lines in the console. It supports colorized printing.
type StatusLinePrinter struct {
	// UseStandardError causes the printer to use standard error for its output
	// instead of standard output (the default).
	UseStandardError bool
	// populated indicates whether or not the printer has printed any non-empty
	// content to the status line.
	populated bool
}

// Print prints a message to the status line, overwriting any existing content.
// Color escape sequences are supported. Messages will be truncated to a
// platform-dependent maximum length and padded appropriately.
func (p *StatusLinePrinter) Print(message string) {
	// Determine the output stream to use. We print to color-supporting output
	// streams to ensure that color escape sequences are properly handled.
	output := color.Output
	if p.UseStandardError {
		output = color.Error
	}

	// Print the message.
	fmt.Fprintf(output, statusLineFormat, message)

	// Update our populated status. The line is always populated in this case
	// because even an empty message will be padded with spaces.
	// TODO: We could possibly make this more precise, e.g. tracking whether or
	// not message is empty or contains only spaces. In cases like these,
	// BreakIfPopulated could potentially just return the cursor to the beginning
	// of the line instead of printing a newline. But it's a bit unclear what
	// the semantics of this should look like, what types of whitespace should
	// be classified as empty, etc. For example, an empty line might be used as
	// a visual delimiter, or a message could contain tabs and/or newlines.
	p.populated = true
}

// Clear clears any content on the status line and moves the cursor back to the
// beginning of the line.
func (p *StatusLinePrinter) Clear() {
	// Determine the output stream to use.
	output := os.Stdout
	if p.UseStandardError {
		output = os.Stderr
	}

	// Wipe out any existing content and return the cursor to the beginning of
	// the line.
	fmt.Fprintf(output, statusLineClearFormat, "")

	// Update our populated status.
	p.populated = false
}

// BreakIfPopulated prints a newline character if the current line is non-empty.
func (p *StatusLinePrinter) BreakIfPopulated() {
	// Only perform an operation if the status line is populated with content.
	if p.populated {
		// Determine the output stream to use.
		output := os.Stdout
		if p.UseStandardError {
			output = os.Stderr
		}

		// Print a line break.
		fmt.Fprintln(output)

		// Update our populated status.
		p.populated = false
	}
}

// StatusLinePrompter adapts a StatusLinePrinter to act as a Mutagen prompter.
// The printer will be used to perform messaging and PromptCommandLine will be
// used to perform prompting.
type StatusLinePrompter struct {
	// Printer is the underlying printer.
	Printer *StatusLinePrinter
}

// Message implements prompting.Prompter.Message.
func (p *StatusLinePrompter) Message(message string) error {
	// Print the message.
	p.Printer.Print(message)

	// Success.
	return nil
}

// Prompt implements prompting.Prompter.Prompt.
func (p *StatusLinePrompter) Prompt(message string) (string, error) {
	// If there's any existing content in the printer, then keep it in place and
	// start a new line of output. We do this (as opposed to clearing the line)
	// because that content most likely provides some context for the prompt.
	//
	// HACK: This is somewhat of a heuristic that relies on knowledge of how
	// Mutagen's internal prompting/messaging works in practice.
	p.Printer.BreakIfPopulated()

	// Perform command line prompting.
	//
	// TODO: Should we respect the printer's UseStandardError field here? The
	// gopass package (used by the prompting package) doesn't provide a way to
	// specify its output stream, so there's not a trivial way to implement it,
	// but in practice the UseStandardError field is only used for daemon
	// auto-start output to avoid corrupting output streams in formatted list
	// and monitor commands (which won't generate prompts), so there's no case
	// at the moment where ignoring the UseStandardError setting causes issues.
	return prompting.PromptCommandLine(message)
}
