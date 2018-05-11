package url

import (
	"github.com/pkg/errors"
)

func (p Protocol) supported() bool {
	switch p {
	case Protocol_Local:
		return true
	case Protocol_SSH:
		return true
	default:
		return false
	}
}

func (u *URL) EnsureValid() error {
	// Ensure that the URL is non-nil.
	if u == nil {
		return errors.New("nil URL")
	}

	// Ensure that the protocol is supported.
	if !u.Protocol.supported() {
		return errors.New("unsupported or unknown protocol")
	}

	// Success.
	return nil
}
