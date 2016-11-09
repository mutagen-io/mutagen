package daemon

import (
	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/agent"
	"github.com/havoc-io/mutagen/session"
)

func NewServer() (*grpc.Server, chan struct{}, error) {
	// Create an empty server.
	server := grpc.NewServer()

	// Create and register the daemon service.
	daemonService, termination := NewService()
	RegisterDaemonServer(server, daemonService)

	// Create and register the agent service.
	agentService := agent.NewService()
	agent.RegisterPromptServer(server, agentService)

	// Create and register the session service.
	sessionManager, err := session.NewManager(agentService)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to creation session manager")
	}
	session.RegisterManagerServer(server, sessionManager)

	// Success.
	return server, termination, nil
}
