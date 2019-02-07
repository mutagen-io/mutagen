package filesystem

import (
	"github.com/pkg/errors"
)

// ErrUnsupportedRootType indicates that the filesystem entry at the specified
// path is not supported as a traversal root.
var ErrUnsupportedRootType = errors.New("unsupported root type")
