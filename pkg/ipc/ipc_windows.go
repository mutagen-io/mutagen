package ipc

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

// DialTimeout attempts to establish an IPC connection, timing out after the
// specified duration.
func DialTimeout(path string, timeout time.Duration) (net.Conn, error) {
	// Read the pipe name.
	pipeNameBytes, err := ioutil.ReadFile(path)
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

// listener implements net.Listener but provides additional cleanup facilities
// on top of those provided by the underlying named pipe listener.
type listener struct {
	// Listener is the underlying named pipe listener.
	net.Listener
	// path is the path to the file where the named pipe name is stored.
	path string
}

// Close closes the listener and removes the pipe name record.
func (l *listener) Close() error {
	// Remove the pipe name record.
	if err := os.Remove(l.path); err != nil {
		l.Listener.Close()
		return errors.Wrap(err, "unable to remove pipe name record")
	}

	// Close the underlying listener.
	return l.Listener.Close()
}

// NewListener creates a new IPC listener. It will overwrite any existing pipe
// name record, so an external mechanism should be used to coordinate the
// establishment of listeners.
func NewListener(path string) (net.Listener, error) {
	// Create a unique pipe name.
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate UUID for named pipe")
	}
	pipeName := fmt.Sprintf(`\\.\pipe\mutagen-ipc-%s`, randomUUID.String())

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
	//  SDDL: https://msdn.microsoft.com/en-us/library/windows/desktop/aa379570(v=vs.85).aspx
	//  ACEs: https://msdn.microsoft.com/en-us/library/windows/desktop/aa374928(v=vs.85).aspx
	//  SIDs: https://msdn.microsoft.com/en-us/library/windows/desktop/aa379602(v=vs.85).aspx
	securityDescriptor := fmt.Sprintf("D:P(A;;GA;;;%s)", sid)

	// Create the pipe configuration.
	configuration := &winio.PipeConfig{
		SecurityDescriptor: securityDescriptor,
	}

	// Create the named pipe listener.
	rawListener, err := winio.ListenPipe(pipeName, configuration)
	if err != nil {
		return nil, err
	}

	// Write the pipe name record.
	if err = filesystem.WriteFileAtomic(path, []byte(pipeName), 0600); err != nil {
		rawListener.Close()
		return nil, errors.Wrap(err, "unable to record pipe name")
	}

	// Success.
	return &listener{rawListener, path}, nil
}
