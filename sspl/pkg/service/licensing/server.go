//go:build sspl

// Copyright (c) 2023-present Mutagen IO, Inc.
//
// This program is free software: you can redistribute it and/or modify it under
// the terms of the Server Side Public License, version 1, as published by
// MongoDB, Inc.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE. See the Server Side Public License for more
// details.
//
// You should have received a copy of the Server Side Public License along with
// this program. If not, see
// <http://www.mongodb.com/licensing/server-side-public-license>.

package licensing

import (
	"context"
	"fmt"

	"github.com/mutagen-io/mutagen/sspl/pkg/licensing"
)

// Server provides an implementation of the Licensing service.
type Server struct {
	// UnimplementedLicensingServer is the required base implementation.
	UnimplementedLicensingServer
	// manager is the underlying license manager.
	manager *licensing.Manager
}

// NewServer creates a new licensing server. The licensing service should be
// initialized before a server is created.
func NewServer(manager *licensing.Manager) *Server {
	return &Server{
		manager: manager,
	}
}

// Activate activates a Mutagen Pro license.
func (s *Server) Activate(ctx context.Context, request *ActivateRequest) (*ActivateResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Attempt to set the API key.
	if err := s.manager.SetKey(ctx, request.Key); err != nil {
		return nil, fmt.Errorf("unable to set key: %w", err)
	}

	// TODO: Should we attempt some sort of polling here to check the result of
	// the license setting operation? Or perhaps make SetKey synchronous?

	// Success.
	return &ActivateResponse{}, nil
}

// Status returns Mutagen Pro license status information.
func (s *Server) Status(ctx context.Context, request *StatusRequest) (*StatusResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Poll for state information.
	_, state, err := s.manager.Poll(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to poll for state information: %w", err)
	}

	// Success.
	return &StatusResponse{State: state}, nil
}

// Deactivate deactivates a Mutagen Pro license.
func (s *Server) Deactivate(ctx context.Context, request *DeactivateRequest) (*DeactivateResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Attempt to clear license information.
	if err := s.manager.SetKey(ctx, ""); err != nil {
		return nil, fmt.Errorf("unable to clear license information: %w", err)
	}

	// TODO: Should we attempt some sort of polling here to check the result of
	// the license clearing operation? Or perhaps make SetKey synchronous?

	// Success.
	return &DeactivateResponse{}, nil
}
