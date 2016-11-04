package session

import (
	"golang.org/x/net/context"
)

type Manager struct {
}

func NewManager() (*Manager, error) {
	// TODO: Implement.
	panic("not implemented")
}

func (m *Manager) Start(stream Manager_StartServer) error {
	// TODO: Implement.
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
