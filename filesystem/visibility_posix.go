// +build !windows

package filesystem

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func MarkHidden(path string) error {
	if strings.IndexRune(filepath.Base(path), '.') != 0 {
		return errors.New("only dot-prefixed files are hidden on POSIX")
	}
	return nil
}
