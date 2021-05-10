package api

import (
	"net/http"
)

// AddSecurityHeaders adds a standard set of security headers suitable for APIs.
func AddSecurityHeaders(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Grab the response header map.
		headers := w.Header()

		// Disable MIME type sniffing.
		headers.Set("X-Content-Type-Options", "nosniff")

		// Set a strict Content Security Policy that disables content fetches
		// from all sources and forbids embedding in frames.
		headers.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Use an alternative mechanism for denying embedding in frames. This
		// supports older browsers that don't support CSP2.
		headers.Set("X-Frame-Options", "DENY")

		// Call the underlying handler.
		handler.ServeHTTP(w, r)
	})
}
