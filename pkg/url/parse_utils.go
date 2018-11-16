package url

import (
	"errors"
	"fmt"
)

const escapeRune = '\\'

// split raw string at first at rune found until breakOn rune is found.
// It returns the string before at rune, the string between at and breakOn rune, and an error if encountered.
// If keepAt is true, at rune will be kept inside the second returned value.
// It supports escape character '\' for at, breakOn and '\' runes.
func splitAndBreak(raw string, at rune, breakOn rune, keepAt bool, checkEmpty string) (string, string, error) {
	var split string
	var escape bool

	for i, r := range raw {
		if r == escapeRune && !escape {
			escape = true
			continue
		}

		if escape {
			escape = false
		} else if r == breakOn {
			return "", raw, nil
		} else if r == at {
			rawIndex := i
			if !keepAt {
				rawIndex += 1
			}
			raw = raw[rawIndex:]
			var err error = nil
			if len(split) == 0 && len(checkEmpty) > 0 {
				err = errors.New(fmt.Sprintf("empty %s specified", checkEmpty))
			}
			return split, raw, err
		}

		split += string(r)
	}

	return "", raw, nil
}
