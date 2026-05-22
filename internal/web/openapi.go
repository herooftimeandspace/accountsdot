package web

import "net/http"

// handleOpenAPISpec serves the generated API contract to local clients and
// tests through NewAppHandler. The payload is generated from
// docs/api/openapi-source.json, does not inspect request state, and exists so
// frontend/runtime clients can discover which routes are DEV mocks, accepted
// no-op placeholders, or DB-backed/planned callable APIs.
func handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(openAPISpecJSON))
}
