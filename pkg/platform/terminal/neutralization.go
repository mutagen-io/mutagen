package terminal

import (
	"strings"
)

// controlCharacterNeutralizer is a string replacer that terminal neutralizes
// control characters.
var controlCharacterNeutralizer = strings.NewReplacer(
	"\x1b", "^[",
	"\r", "\\r",
)

// NeutralizeControlCharacters returns a copy of a string with any terminal
// control characters neutralized.
func NeutralizeControlCharacters(value string) string {
	return controlCharacterNeutralizer.Replace(value)
}
