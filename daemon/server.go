package daemon

import (
	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/session"
	"github.com/havoc-io/mutagen/ssh"
)

func NewServer() (*grpc.Server, chan struct{}, error) {
	// Create an empty server.
	server := grpc.NewServer()

	// Create and register the daemon service.
	daemonService, termination := NewService()
	RegisterDaemonServer(server, daemonService)

	// Create and register the SSH service.
	sshService := ssh.NewService()
	ssh.RegisterPromptServer(server, sshService)

	// Create and register the session service.
	sessionService, err := session.NewService()
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create session service")
	}
	session.RegisterSessionsServer(server, sessionService)

	// Success.
	return server, termination, nil
}
