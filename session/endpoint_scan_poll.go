// +build ignore

package session

import (
	"context"
	"time"
)

const (
	scanPollInterval      = 10 * time.Second
	watchEventsBufferSize = 10
)

func watch(_ context.Context, _ string, _ chan struct{}) error {
	return nil
}
