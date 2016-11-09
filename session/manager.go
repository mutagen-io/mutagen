package session

import (
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/agent"
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

func (m *Manager) Start(stream Manager_StartServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive start request")
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
		Session:             session,
		SynchronizationState: &SynchronizationState{},
	}

	// TODO: Implement.
	_ = sessionState
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
