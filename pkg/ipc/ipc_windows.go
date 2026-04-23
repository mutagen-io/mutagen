package ipc

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/user"

	"github.com/google/uuid"
	"github.com/mutagen-io/mutagen/pkg/logging"

	"github.com/Microsoft/go-winio"

	"github.com/mutagen-io/mutagen/pkg/must"
)

// DialContext attempts to establish an IPC connection, timing out if the
// provided context expires.
func DialContext(ctx context.Context, path string) (net.Conn, error) {
	// Read the pipe name.
	pipeNameBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read pipe name: %w", err)
	}
	pipeName := string(pipeNameBytes)

	// Attempt to connect.
	return winio.DialPipeContext(ctx, pipeName)
}

// listener implements net.Listener but provides additional cleanup facilities
// on top of those provided by the underlying named pipe listener.
type listener struct {
	// Listener is the underlying named pipe listener.
	net.Listener
	// path is the path to the file where the named pipe name is stored.
	path string

	logger *logging.Logger
}

// Close closes the listener and removes the pipe name record.
func (l *listener) Close() error {
	// Remove the pipe name record.
	if err := os.Remove(l.path); err != nil {
		must.Close(l.Listener, l.logger)
		return fmt.Errorf("unable to remove pipe name record: %w", err)
	}

	// Close the underlying listener.
	return l.Listener.Close()
}

// NewListener creates a new IPC listener.
func NewListener(path string, logger *logging.Logger) (net.Listener, error) {
	// Create a unique pipe name.
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("unable to generate UUID for named pipe: %w", err)
	}
	pipeName := fmt.Sprintf(`\\.\pipe\mutagen-ipc-%s`, randomUUID.String())

	// Compute the SID of the user.
	user, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("unable to look up current user: %w", err)
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

	// Attempt to create (and open) the endpoint path where we will record the
	// underlying named pipe name. In order to match the semantics of UNIX
	// domain sockets, we enforce that the file doesn't exist. We do this before
	// attempt to create the named pipe to avoid unnecessary overhead.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("unable to open endpoint: %w", err)
	}

	// Defer closure of the endpoint file when we're done, along with removal in
	// the event of failure.
	var successful bool
	defer func() {
		must.Close(file, logger)
		if !successful {
			must.OSRemove(path, logger)
		}
	}()

	// Create the named pipe listener.
	rawListener, err := winio.ListenPipe(pipeName, configuration)
	if err != nil {
		return nil, err
	}

	// Write the pipe name. This isn't 100% atomic since the name could be
	// partially written, but MoveFileEx isn't guaranteed to be atomic either,
	// so renaming a file into place here doesn't help much.
	if _, err := file.Write([]byte(pipeName)); err != nil {
		return nil, fmt.Errorf("unable to write pipe name: %w", err)
	}

	// Mark ourselves as successful.
	successful = true

	// Success.
	return &listener{
		Listener: rawListener,
		path:     path,
		logger:   logger,
	}, nil
}
