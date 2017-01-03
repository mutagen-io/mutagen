package session

import (
	"github.com/havoc-io/mutagen/rsync"
	"github.com/havoc-io/mutagen/sync"
)

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
	BaseSnapshotSignature    []rsync.BlockHash
	ExpectedSnapshotChecksum []byte
}

type scanResponse struct {
	SnapshotChecksum []byte
	SnapshotDelta    []rsync.Operation
	TryAgain         bool
}

type transmitRequest struct {
	Path          string
	BaseSignature []rsync.BlockHash
}

type transmitResponse struct {
	Operation rsync.Operation
}

type stageRequest struct {
	Transitions []sync.Change
}

type stageResponse struct {
	Status StagingStatus
}

type transitionRequest struct {
	Transitions []sync.Change
}

type transitionResponse struct {
	Changes  []sync.Change
	Problems []sync.Problem
}
