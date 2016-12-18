package session

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/ssh"
	"github.com/havoc-io/mutagen/state"
)

const (
	MethodCreate    = "session.Create"
	MethodList      = "session.List"
	MethodPause     = "session.Pause"
	MethodResume    = "session.Resume"
	MethodTerminate = "session.Terminate"
)

type Service struct {
	// sshService performs registration and deregistration of prompters.
	sshService *ssh.Service
	// notifier tracks changes to session states.
	notifier *state.Notifier
	// sessionLock locks the sessions registry.
	sessionsLock sync.Mutex
	// TODO: Add session registry.
}

func NewService(sshService *ssh.Service) (*Service, error) {
	// Create a notifier to track state changes.
	notifier := state.NewNotifier()

	// TODO: Create the session registry.

	// TODO: Load existing sessions.

	// Success.
	return &Service{
		sshService: sshService,
		notifier:   notifier,
	}, nil
}

func (s *Service) Methods() map[string]rpc.Handler {
	return map[string]rpc.Handler{
		MethodCreate:    s.create,
		MethodList:      s.list,
		MethodPause:     s.pause,
		MethodResume:    s.resume,
		MethodTerminate: s.terminate,
	}
}

func (s *Service) Shutdown() error {
	// TODO: Implement.
	return errors.New("not implemented")
}

func (s *Service) create(stream *rpc.HandlerStream) {
	// TODO: Implement.
}

// byCreationDate implements the sort interface for SessionState, sorting
// sessions by creation date. It is used by the List handler.
type byCreationDate []*SessionState

func (d byCreationDate) Len() int {
	return len(d)
}

func (d byCreationDate) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d byCreationDate) Less(i, j int) bool {
	// This comparison relies on the fact that Nanos can't be negative (at least
	// not according to the Protocol Buffers definition of its value). If Nanos
	// could be negative, we'd have to consider cases where seconds were equal
	// or within 1 of each other.
	return d[i].Session.CreationTime.Seconds < d[j].Session.CreationTime.Seconds ||
		(d[i].Session.CreationTime.Seconds == d[j].Session.CreationTime.Seconds &&
			d[i].Session.CreationTime.Nanos < d[j].Session.CreationTime.Nanos)
}

func (s *Service) list(stream *rpc.HandlerStream) {
	// TODO: Implement.
}

func (s *Service) pause(stream *rpc.HandlerStream) {
	// TODO: Implement.
}

func (s *Service) resume(stream *rpc.HandlerStream) {
	// TODO: Implement.
}

func (s *Service) terminate(stream *rpc.HandlerStream) {
	// TODO: Implement.
}
