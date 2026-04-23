package staging

import (
	"hash"
	"io"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/synchronization/endpoint/local/staging/store"
)

// Stager is an implementation of local.stager that uses a content-addressable
// store to stage files.
type Stager struct {
	// store is the stager's underlying store.
	store  *store.Store
	logger *logging.Logger
}

// NewStager creates a new stager.
func NewStager(root string, hideRoot bool, maximumFileSize uint64, hasherFactory func() hash.Hash, logger *logging.Logger) *Stager {
	return &Stager{
		store:  store.NewStore(root, hideRoot, maximumFileSize, hasherFactory, logger),
		logger: logger,
	}
}

// Initialize implements local.stager.Initialize.
func (s *Stager) Initialize() error {
	return s.store.Initialize()
}

// Contains implements local.stager.Contains.
func (s *Stager) Contains(path string, digest []byte) (bool, error) {
	return s.store.Contains(path, digest)
}

// Sink implements rsync.Sinker.Sink.
func (s *Stager) Sink(path string) (io.WriteCloser, error) {
	storage, err := s.store.Allocate(s.logger)
	if err != nil {
		return nil, err
	}
	return &Sink{path, storage}, nil
}

// Provide implements core.Provider.Provide.
func (s *Stager) Provide(path string, digest []byte) (string, error) {
	return s.store.Path(path, digest)
}

// Finalize implements local.stager.Finalize.
func (s *Stager) Finalize() error {
	return s.store.Finalize()
}

// Sink implements io.WriterCloser for Stager's Sink method.
type Sink struct {
	// path is the path associated with the sink.
	path string
	// storage is the underlying file storage.
	storage *store.Storage
}

// Writer implements io.Writer.Write.
func (s *Sink) Write(data []byte) (int, error) {
	return s.storage.Write(data)
}

// Close implements io.Closer.Close.
func (s *Sink) Close() error {
	return s.storage.Commit(s.path)
}
