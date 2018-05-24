package session

import (
	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/sync"
)

type initializeRequest struct {
	Session     string
	Version     Version
	Root        string
	Ignores     []string
	SymlinkMode sync.SymlinkMode
	Alpha       bool
}

type initializeResponse struct {
	PreservesExecutability bool
	Error                  string
}

type pollRequest struct{}

type pollCompletionRequest struct{}

type pollResponse struct {
	Error string
}

type scanRequest struct {
	BaseSnapshotSignature rsync.Signature
}

type scanResponse struct {
	TryAgain      bool
	SnapshotDelta []rsync.Operation
	Error         string
}

type stageRequest struct {
	Paths   []string
	Entries []*sync.Entry
}

type stageResponse struct {
	Paths      []string
	Signatures []rsync.Signature
	Error      string
}

type supplyRequest struct {
	Paths      []string
	Signatures []rsync.Signature
}

type transitionRequest struct {
	Transitions []*sync.Change
}

type transitionResponse struct {
	Changes  []*sync.Change
	Problems []*sync.Problem
	Error    string
}

type endpointRequest struct {
	Poll       *pollRequest
	Scan       *scanRequest
	Stage      *stageRequest
	Supply     *supplyRequest
	Transition *transitionRequest
}
