package daemon

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/julienschmidt/httprouter"

	"github.com/mutagen-io/mutagen/pkg/api"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

// Service is the daemon service implementation.
type Service struct {
	// done is the termination signaling channel.
	done chan struct{}
	// doneOnce guards closure of done.
	doneOnce sync.Once
}

// NewService creates a new daemon service instance.
func NewService() *Service {
	return &Service{
		done: make(chan struct{}),
	}
}

// Done returns a channel that is closed after a client requests termination of
// the daemon. Successive calls return the same channel.
func (s *Service) Done() <-chan struct{} {
	return s.done
}

// Register registers service endpoints with the specified router.
func (s *Service) Register(router *httprouter.Router) {
	router.HandlerFunc(http.MethodGet, "/api/v0/daemon", s.get)
	router.HandlerFunc(http.MethodDelete, "/api/v0/daemon", s.delete)
}

// getResponse is the response type returned by metadata.
type getResponse struct {
	// VersionMajor is the major component of the daemon version.
	VersionMajor uint64 `json:"versionMajor"`
	// VersionMinor is the minor component of the daemon version.
	VersionMinor uint64 `json:"versionMinor"`
	// VersionPatch is the patch component of the daemon version.
	VersionPatch uint64 `json:"versionPatch"`
	// VersionTag is the tag component of the daemon version.
	VersionTag string `json:"versionTag,omitempty"`
}

// get handles GET requests and serves daemon metadata.
func (s *Service) get(w http.ResponseWriter, r *http.Request) {
	// Set the response content type.
	api.SetContentTypeJSON(w)

	// Create and encode the response.
	json.NewEncoder(w).Encode(&getResponse{
		VersionMajor: mutagen.VersionMajor,
		VersionMinor: mutagen.VersionMinor,
		VersionPatch: mutagen.VersionPatch,
		VersionTag:   mutagen.VersionTag,
	})
}

// delete handles DELETE requests and signals daemon termination.
func (s *Service) delete(w http.ResponseWriter, r *http.Request) {
	// Indicate to the client that the request has been accepted. We do this
	// before signaling termination so that we can ensure the client receives a
	// response before the daemon terminates (assuming it heeds the request).
	w.WriteHeader(http.StatusAccepted)

	// Signal termination.
	s.doneOnce.Do(func() {
		close(s.done)
	})
}
