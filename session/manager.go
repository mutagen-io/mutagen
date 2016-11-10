package session

import (
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	uuid "github.com/satori/go.uuid"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/agent"
	"github.com/havoc-io/mutagen/url"
)

type Manager struct {
	agentService *agent.Service
	stateChange  *sync.Cond
	sessions     map[string]*SessionState
}

func NewManager(agentService *agent.Service) (*Manager, error) {
	// Create the state change condition variable.
	stateChange := sync.NewCond(&sync.Mutex{})

	// TODO: Load existing sessions and start their synchronization loops if
	// they aren't paused.
	sessions := make(map[string]*SessionState)

	// Success.
	return &Manager{
		agentService: agentService,
		stateChange:  stateChange,
		sessions:     sessions,
	}, nil
}

func (m *Manager) clientAndPathForURL(raw string, prompter agent.Prompter) (*grpc.ClientConn, string, error) {
	// Handle based on URL type.
	if urlType := url.Classify(raw); urlType == url.TypePath {
		client, err := m.agentService.ConnectLocal()
		if err != nil {
			return nil, "", errors.Wrap(err, "unable to create local agent")
		}
		return client, raw, nil
	} else if urlType == url.TypeSSH {
		remote, err := url.ParseSSH(raw)
		if err != nil {
			return nil, "", errors.Wrap(err, "unable to parse SSH URL")
		}
		client, err := m.agentService.ConnectSSH(remote, prompter)
		if err != nil {
			return nil, "", errors.Wrap(err, "unable to create SSH agent connection")
		}
		return client, remote.Path, nil
	}

	// Handle invalid URLs.
	return nil, "", errors.New("invalid URL")
}

func (m *Manager) Start(stream Manager_StartServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive start request")
	}

	// Create a prompter wrapper around the stream.
	prompter := func(prompt *agent.PromptRequest) (*agent.PromptResponse, error) {
		if err := stream.Send(&StartResponse{Prompt: prompt}); err != nil {
			return nil, errors.Wrap(err, "unable to send prompt request")
		} else if response, err := stream.Recv(); err != nil {
			return nil, errors.Wrap(err, "unable to receive prompt response")
		} else {
			return response.Response, err
		}
	}

	// Connect to alpha.
	alpha, alphaPath, err := m.clientAndPathForURL(request.Alpha, prompter)
	if err != nil {
		return errors.Wrap(err, "unable to connect to alpha")
	}

	// Connect to beta.
	beta, betaPath, err := m.clientAndPathForURL(request.Beta, prompter)
	if err != nil {
		alpha.Close()
		return errors.Wrap(err, "unable to connect to beta")
	}

	// Create a session.
	now := time.Now()
	session := &Session{
		Identifier:           uuid.NewV4().String(),
		Version:              SessionVersion_Version1,
		CreationTime:         &now,
		CreatingVersionMajor: mutagen.VersionMajor,
		CreatingVersionMinor: mutagen.VersionMinor,
		CreatingVersionPatch: mutagen.VersionPatch,
		Alpha:                request.Alpha,
		Beta:                 request.Beta,
	}

	// Create the session state.
	sessionState := &SessionState{
		Session:              session,
		SynchronizationState: &SynchronizationState{},
	}

	// TODO: Implement.
	_ = sessionState
	_ = alpha
	_ = alphaPath
	_ = beta
	_ = betaPath
	panic("not implemented")
}

func (m *Manager) List(request *ListRequest, responses Manager_ListServer) error {
	// TODO: Implement.
	panic("not implemented")
}

func (m *Manager) Pause(_ context.Context, request *PauseRequest) (*PauseResponse, error) {
	// TODO: Implement.
	panic("not implemented")
}

// TODO: Add Resolve.

func (m *Manager) Resume(stream Manager_ResumeServer) error {
	// TODO: Implement.
	panic("not implemented")
}

func (m *Manager) Stop(_ context.Context, request *StopRequest) (*StopResponse, error) {
	// TODO: Implement.
	panic("not implemented")
}
