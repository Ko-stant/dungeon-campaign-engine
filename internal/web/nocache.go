// Package web holds HTTP helpers shared by the server's page and asset routes.
package web

import "net/http"

// NoCache makes browsers revalidate every response before reusing it. For
// static files served by http.FileServer that costs a cheap 304 when nothing
// changed, and it stops stale CSS/JS surviving a rebuild.
func NoCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}
