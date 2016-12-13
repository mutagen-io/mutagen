package session

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/ssh"
)

type Service struct {
	// sshService performs registration and deregistration of prompters.
	sshService *ssh.Service
	// TODO: Add notifier.
	// TODO: Add session registry.
}

func NewService(sshService *ssh.Service) (*Service, error) {
	// TODO: Create the notifier.

	// TODO: Create the session registry and load sessions.

	// Success.
	return &Service{
		sshService: sshService,
	}, nil
}

func (s *Service) Shutdown() error {
	// TODO: Implement.
	return errors.New("not implemented")
}

func (s *Service) Start(stream *rpc.HandlerStream) {
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
	return d[i].Session.CreationTime.Before(*d[j].Session.CreationTime)
}

func (s *Service) List(stream *rpc.HandlerStream) {
	// TODO: Implement.
}

func (s *Service) Pause(stream *rpc.HandlerStream) {
	// TODO: Implement.
}

func (s *Service) Resume(stream *rpc.HandlerStream) {
	// TODO: Implement.
}

func (s *Service) Stop(stream *rpc.HandlerStream) {
	// TODO: Implement.
}
