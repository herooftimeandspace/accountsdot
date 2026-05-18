package web

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// NewAppHandler builds the value used by internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func NewAppHandler(deps HealthDependencies) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(handleIndex))
	mux.Handle("/sync-dashboard", http.HandlerFunc(handleSyncDashboard))
	mux.Handle("/sync-dashboard/mappings", http.HandlerFunc(handleSyncDashboardMappings))
	mux.Handle("/metrics", http.HandlerFunc(handleMetrics))
	mux.Handle("/events/stream", http.HandlerFunc(handleEventStream))
	mux.Handle("/api/v1/session/me", http.HandlerFunc(handleSessionMe))
	mux.Handle("/api/v1/workflows", http.HandlerFunc(handleWorkflowList))
	mux.Handle("/api/v1/workflows/", http.HandlerFunc(handleWorkflowRoutes))
	mux.Handle("/api/v1/approvals", http.HandlerFunc(handleApprovalList))
	mux.Handle("/api/v1/approvals/", http.HandlerFunc(handleApprovalRoutes))
	mux.Handle("/api/v1/sync-status/", http.HandlerFunc(handleSyncStatusRoutes))
	mux.Handle("/api/v1/room-mappings", http.HandlerFunc(handleRoomMappings))
	mux.Handle("/api/v1/annual-reset", http.HandlerFunc(handleAnnualReset))
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

	health := NewHealthHandler(deps)
	mux.Handle("/health", health)
	mux.Handle("/health/live", health)
	mux.Handle("/health/ready", health)

	return mux
}

// handleIndex handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
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

// handleSyncDashboard handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
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

// handleSyncDashboardMappings handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
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

// handleMetrics handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte("# TYPE app_up gauge\napp_up 1\n"))
}

// handleEventStream handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func handleEventStream(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	_, _ = w.Write([]byte("event: ready\ndata: {\"status\":\"connected\"}\n\n"))
}

// handleSessionMe handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func handleSessionMe(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": false,
		"mode":          "edge-auth-proxy",
	})
}

// handleWorkflowList handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func handleWorkflowList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": []any{},
	})
}

// handleWorkflowRoutes handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
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

// handleApprovalList handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func handleApprovalList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": []any{},
	})
}

// handleApprovalRoutes handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
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

// handleSyncStatusRoutes handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
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

// handleRoomMappings handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
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

// handleAnnualReset handles the request path for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
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

// writeSyncTabResponse writes the response payload for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
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

// writeJSON writes the response payload for internal/web/app.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// registerDevOffboardingRoutes keeps the Offboarding route registration count
// stable while issue #42 splits security-risk account review into Reports. The
// registered handlers are both read-only GET page APIs; the existing
// /api/v1/dev/offboarding/records/{id}/end-date mutation remains registered in
// NewAppHandler so reviewers can see the only Offboarding write boundary next
// to the route inventory in docs/external-write-inventory.md.
func registerDevOffboardingRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/dev/pages/offboarding", http.HandlerFunc(handleDevOffboardingPage))
	mux.Handle("/api/v1/dev/pages/reports/security-issues", http.HandlerFunc(handleDevSecurityIssuesReportPage))
}
