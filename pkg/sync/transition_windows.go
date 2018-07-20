package sync

import (
	"strings"
)

func pathContainsInvalidCharacters(path string) bool {
	return strings.Contains(path, "\\")
}
