package environment

import (
	"strings"

	"github.com/pkg/errors"
)

// TODO: When documenting this function, document that it ignores lines
// beginning with '=' (i.e. it ignores variable specifications with empty
// variable names).
func Parse(environment []string) (map[string]string, error) {
	// Create the result.
	result := make(map[string]string, len(environment))

	// Process each line.
	for _, e := range environment {
		// If the line specifies an empty variable name, ignore it. On Windows,
		// these are vestigial hacks that exist to enable some level of MS-DOS
		// compatibility. On POSIX, these aren't really valid, even though they
		// can be set (though they typically can't be unset). Since they are
		// never relevant to our code, we ignore them. In fact, it's essential
		// that we ignore them for our tests because Windows will usually
		// specify multiple values that start with '=' (e.g. =::=::\ and
		// =C:=C:\Users\username\value) and will actually treat the variable
		// name as starting with '=' (so calling os.Getenv("=::") will yield
		// "::\"). Note that these won't appear in cmd.exe via the "set"
		// command, and only some will appear via mintty's "env" implementation,
		// usually mangled (e.g. "=::" becomes "!::"). Instead, they'll only
		// really appear in programs via GetEnvironmentStrings, which is what Go
		// uses to grab the environment variable block on Windows.
		if len(e) > 0 && e[0] == '=' {
			continue
		}

		// Split the line on the first equal sign. If the line doesn't
		components := strings.SplitN(e, "=", 2)

		// If the line doesn't have the requisite number of components, then
		// it's invalid.
		if len(components) != 2 {
			return nil, errors.Errorf("invalid variable specification: %s", e)
		}

		// Store the variable's value.
		result[components[0]] = components[1]
	}

	// Success.
	return result, nil
}

// TODO: When documenting this function, make a note that it's designed to be
// platform-agnostic, since we use it on remotes as well, and that's why it does
// the newline replacement.
func ParseBlock(environment string) (map[string]string, error) {
	// Convert line endings, trim trailing newlines, and split the output into
	// individual lines.
	environment = strings.Replace(environment, "\r\n", "\n", -1)
	environment = strings.TrimSpace(environment)
	lines := strings.Split(environment, "\n")

	// Call the base parse function.
	return Parse(lines)
}
