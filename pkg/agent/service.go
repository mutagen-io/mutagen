// +build !agent

package agent

import (
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

// Service is the agent service implementation.
type Service struct{}

// NewService creates a new agent service instance.
func NewService() *Service {
	return &Service{}
}

// Register registers service endpoints with the specified router.
func (s *Service) Register(router *httprouter.Router) {
	router.Handle(http.MethodGet, "/api/v0/agents/:goos/:goarch", s.get)
}

// get handles GET requests and serves agent executables.
func (s *Service) get(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Extract the target platform parameters and load the executable stream for
	// the requested platform and defer its closure.
	goos := ps.ByName("goos")
	goarch := ps.ByName("goarch")
	executable, err := executableStreamForPlatform(goos, goarch)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	defer executable.Close()

	// Set the response content type based on the target platform.
	if goos == "windows" {
		w.Header().Set("Content-Type", "application/x-dosexec")
	} else if goos == "darwin" {
		w.Header().Set("Content-Type", "application/x-mach-binary")
	} else if goos == "aix" {
		w.Header().Set("Content-Type", "application/x-coffexec")
	} else {
		w.Header().Set("Content-Type", "application/x-executable")
	}

	// Write the executable stream.
	io.Copy(w, executable)
}
