package url

import (
	"errors"
	"fmt"
)

// split raw string at first at rune found until breakOn rune is found.
// It returns the string before at rune, the string between at and breakOn rune, and an error if encountered.
// If keepAt is true, at rune will be kept inside the second returned value.
func splitAndBreak(raw string, at rune, breakOn rune, keepAt bool, checkNonEmptyField string) (string, string, error) {
	var split = ""
	for i, r := range raw {
		if r == breakOn {
			break
		} else if r == at {
			split = raw[:i]
			rawIndex := i
			if !keepAt {
				rawIndex += 1
			}
			raw = raw[rawIndex:]
			if i == 0 && len(checkNonEmptyField) > 0 {
				return split, raw, errors.New(fmt.Sprintf("empty %s specified", checkNonEmptyField))
			}
			break
		}
	}
	return split, raw, nil
}