package session

import (
	"github.com/havoc-io/mutagen/rsync"
	"github.com/havoc-io/mutagen/sync"
)

type initializeRequest struct {
	Session string
	Version Version
	Root    string
	Alpha   bool
	Ignores []string
}

type initializeResponse struct {
	PreservesExecutability bool
	Error                  string
}

type scanRequest struct {
	BaseSnapshotSignature    []rsync.BlockHash
	ExpectedSnapshotChecksum []byte
}

type scanResponse struct {
	Delta []rsync.Operation
	Error string
}

type transmitRequest struct {
	Path          string
	BaseSignature []rsync.BlockHash
}

type transmitResponse struct {
	Operation rsync.Operation
	Error     string
}

type applyRequest struct {
	Transitions []sync.Change
}

type applyResponse struct {
	Status   StagingStatus
	Done     bool
	Changes  []sync.Change
	Problems []sync.Problem
	Error    string
}
