// +build !windows

package filesystem

import (
	"os"
)

func watchRootParametersEqual(first, second os.FileInfo) bool {
	return os.SameFile(first, second)
}
