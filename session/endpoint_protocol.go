package session

import (
	"github.com/havoc-io/mutagen/rsync"
	"github.com/havoc-io/mutagen/sync"
)

const (
	endpointChannelControl uint8 = iota
	endpointChannelWatchEvents
	endpointChannelRsyncUpdates
	endpointChannelRsyncClient
	endpointChannelRsyncServer
	numberOfEndpointChannels
)

type watchEvent struct{}

type initializeRequest struct {
	Session string
	Version Version
	Root    string
	Ignores []string
	Alpha   bool
}

type initializeResponse struct {
	PreservesExecutability bool
}

type scanRequest struct {
	BaseSnapshotSignature rsync.Signature
}

type scanResponse struct {
	TryAgain      bool
	SnapshotDelta []rsync.Operation
}

type stageRequest struct {
	Transitions []sync.Change
}

type stageResponse struct{}

type transitionRequest struct {
	Transitions []sync.Change
}

type transitionResponse struct {
	Changes  []sync.Change
	Problems []sync.Problem
}

type endpointRequest struct {
	Scan       *scanRequest
	Stage      *stageRequest
	Transition *transitionRequest
}
