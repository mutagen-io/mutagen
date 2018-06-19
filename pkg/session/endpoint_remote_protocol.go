package session

import (
	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/sync"
)

type initializeRequest struct {
	Session       string
	Version       Version
	Root          string
	Configuration *Configuration
	Alpha         bool
}

type initializeResponse struct {
	Error string
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
	SnapshotDelta          []rsync.Operation
	PreservesExecutability bool
	Error                  string
	TryAgain               bool
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
	// HACK: We have to use Archive to wrap our Entry results here because gob
	// won't encode a nil pointer in this slice, and the results of transitions
	// may very well be nil. We probably ought to transition to Protocol Buffers
	// for the remote endpoint protocol eventually, if not fully fledged gRPC,
	// but that's going to require converting all of the rsync types to Protocol
	// Buffers, which I'm not quite read to do.
	Results  []*sync.Archive
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
