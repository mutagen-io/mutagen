package api

import (
	"crypto/subtle"
	"net/http"
)

// RequireAuthentication is HTTP middleware that wraps an existing handler and
// requires that the specified token be provided as the username via HTTP basic
// authentication (with an empty password) for all requests. Requests with
// missing or incorrect credentials are rejected with a 401 status code.
func RequireAuthentication(handler http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization.
		username, password, ok := r.BasicAuth()
		authorized := ok &&
			subtle.ConstantTimeCompare([]byte(username), []byte(token)) == 1 &&
			password == ""

		// Handle the request accordingly.
		if authorized {
			handler.ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	})
}
