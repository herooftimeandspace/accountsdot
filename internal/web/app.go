package web

import (
	"encoding/json"
	"net/http"
	"strings"
)

func NewAppHandler(deps HealthDependencies) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(handleIndex))
	mux.Handle("/metrics", http.HandlerFunc(handleMetrics))
	mux.Handle("/events/stream", http.HandlerFunc(handleEventStream))
	mux.Handle("/api/v1/session/me", http.HandlerFunc(handleSessionMe))
	mux.Handle("/api/v1/workflows", http.HandlerFunc(handleWorkflowList))
	mux.Handle("/api/v1/workflows/", http.HandlerFunc(handleWorkflowRoutes))
	mux.Handle("/api/v1/approvals", http.HandlerFunc(handleApprovalList))
	mux.Handle("/api/v1/approvals/", http.HandlerFunc(handleApprovalRoutes))

	health := NewHealthHandler(deps)
	mux.Handle("/health", health)
	mux.Handle("/health/live", health)
	mux.Handle("/health/ready", health)

	return mux
}

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

func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte("# TYPE app_up gauge\napp_up 1\n"))
}

func handleEventStream(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	_, _ = w.Write([]byte("event: ready\ndata: {\"status\":\"connected\"}\n\n"))
}

func handleSessionMe(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": false,
		"mode":          "edge-auth-proxy",
	})
}

func handleWorkflowList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": []any{},
	})
}

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

func handleApprovalList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": []any{},
	})
}

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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
