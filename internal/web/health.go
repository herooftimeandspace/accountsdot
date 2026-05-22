package web

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/provider"
)

const missingRequiredReadinessCheck = "missing_required_check"

type HealthDependencies struct {
	DBReady         func(context.Context) error
	SequenceReady   func(context.Context) error
	ImportPathReady func(context.Context) error
	SFTPReady       func(context.Context) error
	GoogleReady     func(context.Context) error
	ProviderReady   func(context.Context) map[string]string
	GlobalPaused    func(context.Context) (bool, error)
}

type healthResponse struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
	Controls     map[string]string `json:"controls,omitempty"`
}

// NewHealthHandler registers the Phase 0 diagnostic endpoints used by
// NewAppHandler and health tests. It evaluates dependency and global-pause
// callbacks supplied by cmd/provisioner or tests, returns JSON for liveness and
// readiness, and never mutates provider, database, session, or DEV mock state.
func NewHealthHandler(deps HealthDependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		writeHealth(w, http.StatusOK, liveness())
	})
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		snapshot := evaluateHealth(r.Context(), deps)
		status, payload := snapshot.readiness()
		writeHealth(w, status, payload)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		snapshot := evaluateHealth(r.Context(), deps)
		status, payload := snapshot.readiness()
		writeHealth(w, status, payload)
	})
	return mux
}

type healthCheck struct {
	name     string
	check    func(context.Context) error
	required bool
}

type healthSnapshot struct {
	dependencies map[string]string
	controls     map[string]string
	ready        bool
	paused       bool
}

// dependencyChecks returns the fixed readiness-check order shared by JSON
// health responses and Prometheus metrics. Keeping the names bounded prevents
// provider URLs, credentials, or tenant details from becoming labels.
func dependencyChecks(deps HealthDependencies) []healthCheck {
	return []healthCheck{
		{name: "db", check: deps.DBReady, required: true},
		{name: "sequence", check: deps.SequenceReady, required: true},
		{name: "import_path", check: deps.ImportPathReady, required: true},
		{name: "sftp", check: deps.SFTPReady},
		{name: "google", check: deps.GoogleReady, required: true},
	}
}

// evaluateHealth runs the dependency callbacks for /health/ready, /health, and
// /metrics using the request context supplied by the caller. Missing required
// callbacks fail closed as missing_required_check; unwired optional callbacks
// are reported as not_configured. Any failing callback or active global pause
// clears readiness with a bounded public state that does not expose raw driver
// or provider text.
func evaluateHealth(ctx context.Context, deps HealthDependencies) healthSnapshot {
	snapshot := healthSnapshot{
		dependencies: make(map[string]string, len(dependencyChecks(deps))),
		controls:     make(map[string]string, 1),
		ready:        true,
	}

	for _, dependency := range dependencyChecks(deps) {
		if dependency.check == nil {
			if dependency.required {
				snapshot.dependencies[dependency.name] = missingRequiredReadinessCheck
				snapshot.ready = false
				continue
			}
			snapshot.dependencies[dependency.name] = "not_configured"
			continue
		}
		if err := dependency.check(ctx); err != nil {
			snapshot.dependencies[dependency.name] = "unavailable"
			snapshot.ready = false
			continue
		}
		snapshot.dependencies[dependency.name] = "ok"
	}

	if deps.ProviderReady != nil {
		for name, state := range deps.ProviderReady(ctx) {
			dependencyName := "provider_" + name
			snapshot.dependencies[dependencyName] = state
			if !provider.ConfigurationStatusReady(state) {
				snapshot.ready = false
			}
		}
	}

	if deps.GlobalPaused == nil {
		snapshot.controls["global_pause"] = "not_configured"
	} else {
		paused, err := deps.GlobalPaused(ctx)
		if err != nil {
			snapshot.controls["global_pause"] = "unavailable"
			snapshot.ready = false
		} else if paused {
			snapshot.controls["global_pause"] = "paused"
			snapshot.paused = true
			snapshot.ready = false
		} else {
			snapshot.controls["global_pause"] = "ok"
		}
	}

	return snapshot
}

// liveness keeps /health/live independent from database-backed dependency and
// control checks. Kubernetes-style liveness callers only need to know that the
// HTTP process can respond; pause and dependency state belong to readiness and
// metrics so a database outage cannot make the process look dead.
func liveness() healthResponse {
	return healthResponse{Status: "ok"}
}

// readiness converts the evaluated snapshot into the readiness HTTP contract:
// dependency failures are degraded, a clean global pause is paused, and only an
// unpaused system with no failing dependency reports ok.
func (snapshot healthSnapshot) readiness() (int, healthResponse) {
	status := "ok"
	code := http.StatusOK
	if !snapshot.ready {
		code = http.StatusServiceUnavailable
		status = "degraded"
		if snapshot.paused && dependenciesReady(snapshot.dependencies) {
			status = "paused"
		}
	}

	return code, healthResponse{
		Status:       status,
		Dependencies: snapshot.dependencies,
		Controls:     snapshot.controls,
	}
}

// dependenciesReady distinguishes a clean global pause from a dependency
// outage. not_configured is allowed only for optional unwired checks; missing
// required checks use missing_required_check and therefore remain degraded.
func dependenciesReady(dependencies map[string]string) bool {
	for _, state := range dependencies {
		if state != "ok" && state != "not_configured" {
			return false
		}
	}
	return true
}

// writeHealth serializes the already-evaluated health payload for the health
// routes. It only sets response headers and status; debugging should inspect
// evaluateHealth when a dependency, pause, or readiness value is unexpected.
func writeHealth(w http.ResponseWriter, status int, payload healthResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
