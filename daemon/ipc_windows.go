package daemon

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	npipe "gopkg.in/natefinch/npipe.v2"

	"github.com/havoc-io/mutagen/filesystem"
)

const (
	pipeNameRecordName = "daemon.pipe"
)

func DialTimeout(timeout time.Duration) (net.Conn, error) {
	// Compute the path to the pipe name record.
	pipeNameRecordPath, err := subpath(pipeNameRecordName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute pipe name record path")
	}

	// Read the pipe name.
	pipeNameBytes, err := ioutil.ReadFile(pipeNameRecordPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read pipe name")
	}
	pipeName := string(pipeNameBytes)

	// Attempt to connect.
	return npipe.DialTimeout(pipeName)
}

type daemonListener struct {
	net.Listener
	pipeNameRecordPath string
}

func (l *daemonListener) Close() error {
	// HACK: Recover from any panics that arise when the listener is closed.
	// This is necessary because the npipe package uses the CancelIoEx system
	// call, which is only available on Windows Vista+, when closing listeners,
	// thus resulting in a panic on Windows XP. Since this listener will have
	// the same lifespan as the daemon, it's fine to just ignore the panic that
	// arises when the listener is closed, because the resources will be cleaned
	// up anyway.
	defer func() {
		recover()
	}()

	// Remove the pipe name record, if any. We watch for an empty string because
	// we partially initialize the daemon listener at first (to make use of its
	// safe closure functionality in case of errors).
	if l.pipeNameRecordPath != "" {
		os.Remove(l.pipeNameRecordPath)
	}

	// Close the underlying listener.
	return l.Listener.Close()
}

func NewListener() (net.Listener, error) {
	// Create a unique pipe name.
	pipeName := fmt.Sprintf(`\\.\pipe\mutagen-%s`, uuid.NewV4())

	// Compute the path to the pipe name record.
	pipeNameRecordPath, err := subpath(pipeNameRecordName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute pipe name record path")
	}

	// Create the listener and wrap it up.
	rawListener, err := npipe.Listen(pipeName)
	if err != nil {
		return nil, err
	}
	listener := &daemonListener{rawListener, ""}

	// Write the pipe name record. This is safe since the caller should own the
	// daemon lock. In general, the pipe name record will be cleaned up when the
	// listener is closed, but if there's a crash and a stale record exists, it
	// will be replaced here.
	if err = filesystem.WriteFileAtomic(pipeNameRecordPath, []byte(pipeName), 0600); err != nil {
		listener.Close()
		return nil, errors.Wrap(err, "unable to record pipe name")
	}
	listener.pipeNameRecordPath = pipeNameRecordPath

	// Success.
	return listener, nil
}
