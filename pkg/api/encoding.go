package api

import (
	"net/http"
)

// SetContentTypeJSON is a utility function that sets the Content-Type header
// for an HTTP response to application/json.
func SetContentTypeJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}
