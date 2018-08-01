package sync

import (
	"strings"
)

func containsAlternatePathSeparator(name string) bool {
	return strings.IndexByte(name, '\\') >= 0
}
