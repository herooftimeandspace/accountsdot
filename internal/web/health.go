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

func writeHealth(w http.ResponseWriter, status int, payload healthResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
