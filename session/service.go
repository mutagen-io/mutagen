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

type Service struct {
	stateChange *sync.Cond
	sessions    map[string]*SessionState
}

func NewService() (*Service, error) {
	// Create the state change condition variable.
	stateChange := sync.NewCond(&sync.Mutex{})

	// TODO: Load existing sessions and start their synchronization loops if
	// they aren't paused.
	sessions := make(map[string]*SessionState)

	// Success.
	return &Service{
		stateChange: stateChange,
		sessions:    sessions,
	}, nil
}

func clientConnectionAndPathForURL(raw string, prompter string) (*grpc.ClientConn, string, error) {
	// Handle based on URL type.
	if urlType := url.Classify(raw); urlType == url.TypePath {
		// Create an in-memory agent and connection.
		return dialLocal(), raw, nil
	} else if urlType == url.TypeSSH {
		// Parse the SSH URL.
		remote, err := url.ParseSSH(raw)
		if err != nil {
			return nil, "", errors.Wrap(err, "unable to parse SSH URL")
		}

		// Create the SSH client connection.
		client, err := agent.DialSSH(prompter, remote)
		if err != nil {
			return nil, "", errors.Wrap(err, "unable to create SSH agent connection")
		}

		// Success.
		return client, remote.Path, nil
	}

	// Handle invalid URLs.
	return nil, "", errors.New("invalid URL")
}

func (m *Service) Start(_ context.Context, request *StartRequest) (*StartResponse, error) {
	// Connect to alpha.
	alpha, alphaPath, err := clientConnectionAndPathForURL(request.Alpha, request.Prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to alpha")
	}

	// Connect to beta.
	beta, betaPath, err := clientConnectionAndPathForURL(request.Beta, request.Prompter)
	if err != nil {
		alpha.Close()
		return nil, errors.Wrap(err, "unable to connect to beta")
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
	_ = alphaPath
	_ = betaPath
	alpha.Close()
	beta.Close()
	return nil, errors.New("not implemented")
}

func (m *Service) List(request *ListRequest, responses Sessions_ListServer) error {
	// TODO: Implement.
	return errors.New("not implemented")
}

func (m *Service) Pause(_ context.Context, request *PauseRequest) (*PauseResponse, error) {
	// TODO: Implement.
	return nil, errors.New("not implemented")
}

// TODO: Add Resolve.

func (m *Service) Resume(_ context.Context, request *ResumeRequest) (*ResumeResponse, error) {
	// TODO: Implement.
	return nil, errors.New("not implemented")
}

func (m *Service) Stop(_ context.Context, request *StopRequest) (*StopResponse, error) {
	// TODO: Implement.
	return nil, errors.New("not implemented")
}
