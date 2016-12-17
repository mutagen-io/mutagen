// +build !windows,!darwin darwin,!cgo

package session

import (
	"context"
	"time"
)

const (
	scanPollInterval      = 10 * time.Second
	watchEventsBufferSize = 10
)

func watch(_ context.context, _ string, _ chan struct{}) error {
	return nil
}
