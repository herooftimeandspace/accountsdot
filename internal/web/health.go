package web

import (
	"encoding/json"
	"net/http"
)

type HealthDependencies struct {
	DBReady         func() error
	SequenceReady   func() error
	ImportPathReady func() error
	SFTPReady       func() error
	GoogleReady     func() error
}

type healthResponse struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

// NewHealthHandler builds the value used by internal/web/health.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func NewHealthHandler(deps HealthDependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		writeHealth(w, http.StatusOK, healthResponse{Status: "ok"})
	})
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		status, payload := readiness(deps)
		writeHealth(w, status, payload)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status, payload := readiness(deps)
		writeHealth(w, status, payload)
	})
	return mux
}

// readiness documents the data flow for internal/web/health.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func readiness(deps HealthDependencies) (int, healthResponse) {
	checks := map[string]func() error{
		"db":          deps.DBReady,
		"sequence":    deps.SequenceReady,
		"import_path": deps.ImportPathReady,
		"sftp":        deps.SFTPReady,
		"google":      deps.GoogleReady,
	}

	dependencies := make(map[string]string, len(checks))
	ready := true
	for name, check := range checks {
		if check == nil {
			dependencies[name] = "not_configured"
			continue
		}
		if err := check(); err != nil {
			dependencies[name] = err.Error()
			ready = false
			continue
		}
		dependencies[name] = "ok"
	}

	if !ready {
		return http.StatusServiceUnavailable, healthResponse{
			Status:       "degraded",
			Dependencies: dependencies,
		}
	}

	return http.StatusOK, healthResponse{
		Status:       "ok",
		Dependencies: dependencies,
	}
}

// writeHealth writes the response payload for internal/web/health.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func writeHealth(w http.ResponseWriter, status int, payload healthResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
