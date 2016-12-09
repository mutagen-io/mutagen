package url

import (
	"github.com/pkg/errors"
)

const (
	portMax = 1<<16 - 1
)

func (u *URL) Validate() error {
	// Ensure the protocol is supported.
	if u.Protocol < Protocol_Local || u.Protocol > Protocol_SSH {
		return errors.New("unsupported protocol")
	}

	// Validate that username isn't set for local URLs.
	if u.Protocol == Protocol_Local && u.Username != "" {
		return errors.New("non-empty username for local URL")
	}

	// Validate that hostname isn't set for local URLs and is set for SSH URLs.
	if u.Protocol == Protocol_Local && u.Hostname != "" {
		return errors.New("non-empty hostname for local URL")
	} else if u.Protocol == Protocol_SSH && u.Hostname == "" {
		return errors.New("empty hostname for SSH URL")
	}

	// Ensure that the port is in a valid range. Unfortunately Protocol Buffers
	// doesn't offer a uint16 type.
	if u.Port > portMax {
		return errors.New("port value outside allowed range")
	}

	// Ensure that the path is non-empty.
	if u.Path == "" {
		return errors.New("path is empty")
	}

	// Success.
	return nil
}
