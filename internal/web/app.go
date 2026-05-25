package web

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// NewAppHandler wires the production health endpoints, legacy sync-dashboard
// placeholders, and DEV frontend API routes into one mux for cmd/provisioner
// and internal/web tests. DEV routes registered here are mock-only surfaces;
// write-capable handlers must also be listed in docs/planning/external-write-inventory.md.
func NewAppHandler(deps HealthDependencies) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(handleIndex))
	mux.Handle("/sync-dashboard", http.HandlerFunc(handleSyncDashboard))
	mux.Handle("/sync-dashboard/mappings", http.HandlerFunc(handleSyncDashboardMappings))
	mux.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleMetrics(w, r, deps)
	}))
	mux.Handle("/events/stream", http.HandlerFunc(handleEventStream))
	mux.Handle("/api/v1/openapi.json", http.HandlerFunc(handleOpenAPISpec))
	mux.Handle("/api/v1/session/me", http.HandlerFunc(handleSessionMe))
	mux.Handle("/api/v1/workflows", http.HandlerFunc(handleWorkflowList))
	mux.Handle("/api/v1/workflows/", http.HandlerFunc(handleWorkflowRoutes))
	mux.Handle("/api/v1/approvals", http.HandlerFunc(handleApprovalList))
	mux.Handle("/api/v1/approvals/", http.HandlerFunc(handleApprovalRoutes))
	mux.Handle("/api/v1/sync-status/", http.HandlerFunc(handleSyncStatusRoutes))
	mux.Handle("/api/v1/room-mappings", http.HandlerFunc(handleRoomMappings))
	mux.Handle("/api/v1/annual-reset", http.HandlerFunc(handleAnnualReset))
	mux.Handle("/api/v1/breakglass/login", http.HandlerFunc(handleBreakglassLogin))
	mux.Handle("/api/v1/dev/session", http.HandlerFunc(handleDevSession))
	mux.Handle("/api/v1/dev/login", http.HandlerFunc(handleDevLogin))
	mux.Handle("/api/v1/dev/logout", http.HandlerFunc(handleDevLogout))
	mux.Handle("/api/v1/dev/my-profile", http.HandlerFunc(handleDevMyProfile))
	mux.Handle("/api/v1/dev/feature-flags", http.HandlerFunc(handleDevFeatureFlags))
	mux.Handle("/api/v1/dev/feature-flags/", http.HandlerFunc(handleDevFeatureFlag))
	mux.Handle("/api/v1/dev/search", http.HandlerFunc(handleDevGlobalSearch))
	mux.Handle("/api/v1/dev/pages/onboarding", http.HandlerFunc(handleDevOnboardingPage))
	mux.Handle("/api/v1/dev/onboarding/manual-drafts", http.HandlerFunc(handleDevOnboardingManualDrafts))
	mux.Handle("/api/v1/dev/onboarding/manual-drafts/", http.HandlerFunc(handleDevOnboardingManualDraft))
	mux.Handle("/api/v1/dev/onboarding/rows/", http.HandlerFunc(handleDevOnboardingRoomUpdate))
	registerDevOffboardingRoutes(mux)
	mux.Handle("/api/v1/dev/offboarding/records/", http.HandlerFunc(handleDevOffboardingRecord))
	mux.Handle("/api/v1/dev/pages/departing-seniors", http.HandlerFunc(handleDevDepartingSeniorsPage))
	mux.Handle("/api/v1/dev/departing-seniors/records/", http.HandlerFunc(handleDevDepartingSeniorRecord))
	mux.Handle("/api/v1/dev/pages/data-quality", http.HandlerFunc(handleDevDataQualityPage))
	mux.Handle("/api/v1/dev/pages/room-moves", http.HandlerFunc(handleDevRoomMovesPage))
	mux.Handle("/api/v1/dev/pages/room-moves/bulk-draft", http.HandlerFunc(handleDevRoomMovesBulkDraftPage))
	mux.Handle("/api/v1/dev/room-moves/drafts", http.HandlerFunc(handleDevRoomMoveDrafts))
	mux.Handle("/api/v1/dev/room-moves/drafts/", http.HandlerFunc(handleDevRoomMoveDraft))
	mux.Handle("/api/v1/dev/room-moves/completed", http.HandlerFunc(handleDevRoomMoveCompletedJobs))
	mux.Handle("/api/v1/dev/room-moves/completed/", http.HandlerFunc(handleDevRoomMoveCompletedJob))
	mux.Handle("/api/v1/dev/pages/phone-directory/by-person", http.HandlerFunc(handleDevPhoneDirectoryByPersonPage))
	mux.Handle("/api/v1/dev/pages/phone-directory/by-room", http.HandlerFunc(handleDevPhoneDirectoryByRoomPage))
	mux.Handle("/api/v1/dev/pages/phone-directory/by-department", http.HandlerFunc(handleDevPhoneDirectoryByDepartmentPage))
	mux.Handle("/api/v1/dev/pages/reports/zoom-desk-phone-renames", http.HandlerFunc(handleDevZoomDeskPhoneRenamesReportPage))

	health := NewHealthHandler(deps)
	mux.Handle("/health", health)
	mux.Handle("/health/live", health)
	mux.Handle("/health/ready", health)

	return mux
}

// handleIndex serves the root HTML placeholder used by smoke tests and local
// checks before the React DEV frontend is running. It only writes a static page
// response and returns 404 for every non-root path.
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Go Employee Provisioner</title></head>
<body>
<main>
<h1>Go Employee Provisioner</h1>
<p>Mission-critical employee provisioning with HTML status pages, JSON APIs, and resilient orchestration.</p>
</main>
</body>
</html>`))
}

// handleSyncDashboard serves the legacy static sync-dashboard shell for GET
// requests. The page is a read-only placeholder; live DEV dashboard data comes
// from the React frontend APIs registered separately in NewAppHandler.
func handleSyncDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/sync-dashboard" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Sync Transparency Dashboard</title></head>
<body>
<main>
<h1>Sync Transparency Dashboard</h1>
<p>First-class dry-run sync visibility for Aeries, Incident IQ, photo checks, and Zoom validation.</p>
<button type="button">Refresh</button>
<p data-refresh="15">Auto refresh every 15 seconds</p>
<nav>
<a href="#pending">Pending</a>
<a href="#manual">In Progress / Manual Actions</a>
<a href="#completed">Completed</a>
<a href="#history">History</a>
</nav>
<section id="pending"><h2>Pending</h2></section>
<section id="manual"><h2>In Progress / Manual Actions</h2></section>
<section id="completed"><h2>Completed</h2></section>
<section id="history"><h2>History</h2></section>
<table>
<thead>
<tr><th>User</th><th>Current Step</th><th>Issue/Action</th><th>Date</th></tr>
</thead>
<tbody></tbody>
</table>
<script>
setInterval(function () { window.__syncDashboardLastRefresh = Date.now(); }, 15000);
</script>
</main>
</body>
</html>`))
}

// handleSyncDashboardMappings serves the legacy room-mapping HTML shell for GET
// requests. It does not read provider data or persist room mapping decisions;
// JSON room-mapping stubs below own the mock API behavior.
func handleSyncDashboardMappings(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/sync-dashboard/mappings" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Room Mapping Tool</title></head>
<body>
<main>
<h1>Room Mapping Tool</h1>
<p>Resolve Aeries room strings against Incident IQ rooms and assets.</p>
<form><input type="search" name="query" placeholder="Search room"><button type="submit">Search</button></form>
</main>
</body>
</html>`))
}

// handleMetrics exposes Phase 0 Prometheus-compatible health gauges for smoke
// checks and staging observability. It evaluates the same dependency callbacks
// as /health/ready with the request context, emits only bounded non-secret
// labels, and performs no provider, database, or DEV mock mutation.
func handleMetrics(w http.ResponseWriter, r *http.Request, deps HealthDependencies) {
	snapshot := evaluateHealth(r.Context(), deps)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte("# TYPE app_up gauge\napp_up 1\n"))
	_, _ = w.Write([]byte("# TYPE app_ready gauge\n"))
	_, _ = w.Write([]byte("app_ready " + metricBool(snapshot.ready) + "\n"))
	_, _ = w.Write([]byte("# TYPE app_global_pause gauge\n"))
	_, _ = w.Write([]byte("app_global_pause " + metricBool(snapshot.paused) + "\n"))
	_, _ = w.Write([]byte("# TYPE app_dependency_ready gauge\n"))
	for _, dependency := range dependencyChecks(deps) {
		_, _ = w.Write([]byte("app_dependency_ready{name=\"" + dependency.name + "\"} " + metricDependency(snapshot.dependencies[dependency.name]) + "\n"))
	}
	_, _ = w.Write([]byte("# TYPE app_provider_ready gauge\n"))
	for _, name := range providerMetricNames(snapshot.dependencies) {
		_, _ = w.Write([]byte("app_provider_ready{name=\"" + name + "\"} " + metricDependency(snapshot.dependencies["provider_"+name]) + "\n"))
	}
}

// metricBool renders boolean health state in Prometheus gauge form. It keeps
// handleMetrics focused on bounded metric names and avoids string formatting
// branches in the response-writing path.
func metricBool(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

// metricDependency converts JSON dependency states to a 0/1 readiness gauge.
// not_configured is treated as ready for the same local-smoke-test reason as
// /health/ready; concrete callback failures are the only dependency value that
// clears the metric.
func metricDependency(state string) string {
	if dependencyStateReady(state) {
		return "1"
	}
	return "0"
}

func providerMetricNames(dependencies map[string]string) []string {
	names := []string{}
	for name := range dependencies {
		providerName, ok := strings.CutPrefix(name, "provider_")
		if ok {
			names = append(names, providerName)
		}
	}
	sort.Strings(names)
	return names
}

// handleEventStream opens the placeholder server-sent-events stream used by the
// legacy sync-dashboard shell. It emits only a ready event and does not expose
// workflow, provider, or persona data.
func handleEventStream(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	_, _ = w.Write([]byte("event: ready\ndata: {\"status\":\"connected\"}\n\n"))
}

// handleSessionMe returns the legacy edge-auth placeholder session payload.
// The React DEV persona switcher uses /api/v1/dev/session instead, so this
// route remains unauthenticated and intentionally returns no protected data.
func handleSessionMe(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": false,
		"mode":          "edge-auth-proxy",
	})
}

// handleWorkflowList returns an empty workflow collection for the legacy JSON
// API contract. Real workflow projections are not implemented on this route,
// so it avoids fabricating provider-backed lifecycle data.
func handleWorkflowList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": []any{},
	})
}

// handleWorkflowRoutes returns placeholder workflow detail and retry responses
// for legacy API tests. Retry accepts the request shape but only echoes
// acceptance; it does not enqueue provider work or mutate workflow state.
func handleWorkflowRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/workflows/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(path, "/retry") {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		workflowID := strings.TrimSuffix(path, "/retry")
		workflowID = strings.TrimSuffix(workflowID, "/")
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":          "accepted",
			"workflow_run_id": workflowID,
		})
		return
	}
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"workflow_run_id": path,
		"status":          "planned",
		"items":           []any{},
	})
}

// handleApprovalList returns the empty legacy approval collection. Approval
// persistence and assignment are not modeled on this placeholder API.
func handleApprovalList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": []any{},
	})
}

// handleApprovalRoutes validates the legacy approve/reject URL shape and
// returns an accepted echo for tests. It does not persist approval decisions,
// advance workflows, or write to external systems.
func handleApprovalRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/approvals/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	approvalID := parts[0]
	decision := parts[1]
	if decision != "approve" && decision != "reject" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":      "accepted",
		"approval_id": approvalID,
		"decision":    decision,
	})
}

// handleSyncStatusRoutes serves legacy sync-status tab stubs and override
// acknowledgements. Override requests only return the requested user keys; they
// do not change source-system data, DEV stores, or provider workflows.
func handleSyncStatusRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sync-status/")
	switch {
	case path == "pending":
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"tab":   "pending",
			"items": []any{},
		})
	case path == "in-progress":
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"tab":   "in_progress",
			"items": []any{},
		})
	case path == "completed":
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		writeSyncTabResponse(w, "completed", r.URL.Query())
	case path == "history":
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		writeSyncTabResponse(w, "history", r.URL.Query())
	default:
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) != 3 || parts[2] != "override" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":    "accepted",
			"user_type": parts[0],
			"user_id":   parts[1],
		})
	}
}

// handleRoomMappings provides legacy GET/POST room-mapping API placeholders.
// GET echoes the search query with no results, and POST returns accepted without
// storing mappings or writing to IncidentIQ.
func handleRoomMappings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"query": r.URL.Query().Get("query"),
			"items": []any{},
		})
	case http.MethodPost:
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status": "accepted",
		})
	default:
		http.NotFound(w, r)
	}
}

// handleAnnualReset accepts the legacy annual-reset trigger shape for tests.
// The response is an acknowledgement only; archival work and provider writes are
// not started from this placeholder route.
func handleAnnualReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":        "accepted",
		"workflow_type": "annual_reset_archive",
	})
}

// writeSyncTabResponse serializes the legacy completed/history tab response
// using only request-filter echoes and an empty item list. It keeps tab filters
// visible to tests without manufacturing source-system records.
func writeSyncTabResponse(w http.ResponseWriter, tab string, values url.Values) {
	writeJSON(w, http.StatusOK, map[string]any{
		"tab": tab,
		"filters": map[string]string{
			"site_code":   values.Get("site_code"),
			"user_type":   values.Get("user_type"),
			"school_year": values.Get("school_year"),
		},
		"items": []any{},
	})
}

// writeJSON is the shared encoder for small legacy handlers in this file. DEV
// frontend handlers use it only for response serialization; callers decide auth,
// validation, and whether a mock store mutation is allowed before encoding.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// registerDevOffboardingRoutes keeps Offboarding page, search, and scheduling
// routes grouped so permission reviews can compare read APIs with the DEV-only
// mock write boundaries documented in docs/planning/external-write-inventory.md.
func registerDevOffboardingRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/dev/pages/offboarding", http.HandlerFunc(handleDevOffboardingPage))
	mux.Handle("/api/v1/dev/pages/reports/security-issues", http.HandlerFunc(handleDevSecurityIssuesReportPage))
	mux.Handle("/api/v1/dev/offboarding/candidates", http.HandlerFunc(handleDevOffboardingCandidates))
	mux.Handle("/api/v1/dev/offboarding/emergency-deprovision", http.HandlerFunc(handleDevOffboardingEmergencyDeprovision))
	mux.Handle("/api/v1/dev/offboarding/contractor-offboarding", http.HandlerFunc(handleDevOffboardingContractorSchedule))
}
