package daemon

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"time"

	"github.com/pkg/errors"

	"github.com/google/uuid"

	"github.com/Microsoft/go-winio"

	"github.com/havoc-io/mutagen/pkg/filesystem"
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

	// Convert the timeout duration to a pointer. The go-winio library uses a
	// pointer-based duration to indicate the absence of a timeout. This sort of
	// flies in the face of convention (in the net package, a zero-value
	// duration indicates no timeout), but we can adapt.
	var timeoutPointer *time.Duration
	if timeout != 0 {
		timeoutPointer = &timeout
	}

	// Attempt to connect.
	return winio.DialPipe(pipeName, timeoutPointer)
}

type daemonListener struct {
	net.Listener
	pipeNameRecordPath string
}

func (l *daemonListener) Close() error {
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
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate UUID for named pipe")
	}
	pipeName := fmt.Sprintf(`\\.\pipe\mutagen-%s`, randomUUID.String())

	// Compute the path to the pipe name record.
	pipeNameRecordPath, err := subpath(pipeNameRecordName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute pipe name record path")
	}

	// Compute the SID of the user.
	user, err := user.Current()
	if err != nil {
		return nil, errors.Wrap(err, "unable to look up current user")
	}
	sid := user.Uid

	// Create the security descriptor for the pipe. This is constructed using
	// the Security Descriptor Definition Language (SDDL) (the Discretionary
	// Access Control List (DACL) format), where the value in parentheses is an
	// Access Control Entry (ACE) string. The P flag in the DACL prevents
	// inherited permissions. The ACE string in this case grants "Generic All"
	// (GA) permissions to its associated SID. More information can be found
	// here:
	//	SDDL: https://msdn.microsoft.com/en-us/library/windows/desktop/aa379570(v=vs.85).aspx
	//  ACEs: https://msdn.microsoft.com/en-us/library/windows/desktop/aa374928(v=vs.85).aspx
	//  SIDs: https://msdn.microsoft.com/en-us/library/windows/desktop/aa379602(v=vs.85).aspx
	securityDescriptor := fmt.Sprintf("D:P(A;;GA;;;%s)", sid)

	// Create the pipe configuration.
	configuration := &winio.PipeConfig{
		SecurityDescriptor: securityDescriptor,
	}

	// Create the listener and wrap it up.
	rawListener, err := winio.ListenPipe(pipeName, configuration)
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
