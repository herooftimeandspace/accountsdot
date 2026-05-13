package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/web"
)

// TestSyncDashboardHTMLRoutes exercises and documents internal/web/sync_dashboard_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestSyncDashboardHTMLRoutes(t *testing.T) {
	handler := web.NewAppHandler(web.HealthDependencies{})

	tests := []struct {
		path     string
		contains []string
	}{
		{
			path: "/sync-dashboard",
			contains: []string{
				"Sync Transparency Dashboard",
				"Pending",
				"In Progress / Manual Actions",
				"Completed",
				"History",
				"User",
				"Current Step",
				"Issue/Action",
				"Date",
				"Refresh",
			},
		},
		{
			path: "/sync-dashboard/mappings",
			contains: []string{
				"Room Mapping Tool",
				"Incident IQ",
			},
		},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s returned %d, want 200", tc.path, rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
			t.Fatalf("%s content type = %q, want text/html", tc.path, got)
		}
		body := rec.Body.String()
		for _, fragment := range tc.contains {
			if !strings.Contains(body, fragment) {
				t.Fatalf("%s body missing %q; got %q", tc.path, fragment, body)
			}
		}
	}
}

// TestSyncDashboardJSONRoutes exercises and documents internal/web/sync_dashboard_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestSyncDashboardJSONRoutes(t *testing.T) {
	handler := web.NewAppHandler(web.HealthDependencies{})

	tests := []struct {
		method      string
		path        string
		statusCode  int
		contentType string
		contains    []string
	}{
		{method: http.MethodGet, path: "/api/v1/sync-status/pending", statusCode: http.StatusOK, contentType: "application/json", contains: []string{`"tab":"pending"`, `"items":[]`}},
		{method: http.MethodGet, path: "/api/v1/sync-status/in-progress", statusCode: http.StatusOK, contentType: "application/json", contains: []string{`"tab":"in_progress"`}},
		{method: http.MethodGet, path: "/api/v1/sync-status/completed?site_code=HS&user_type=staff&school_year=2026", statusCode: http.StatusOK, contentType: "application/json", contains: []string{`"tab":"completed"`, `"site_code":"HS"`, `"user_type":"staff"`, `"school_year":"2026"`}},
		{method: http.MethodGet, path: "/api/v1/sync-status/history?site_code=MS&user_type=student&school_year=2025", statusCode: http.StatusOK, contentType: "application/json", contains: []string{`"tab":"history"`, `"site_code":"MS"`, `"user_type":"student"`, `"school_year":"2025"`}},
		{method: http.MethodPost, path: "/api/v1/sync-status/staff/12345/override", statusCode: http.StatusAccepted, contentType: "application/json", contains: []string{`"status":"accepted"`, `"user_type":"staff"`, `"user_id":"12345"`}},
		{method: http.MethodGet, path: "/api/v1/room-mappings?query=rm-101", statusCode: http.StatusOK, contentType: "application/json", contains: []string{`"query":"rm-101"`, `"items":[]`}},
		{method: http.MethodPost, path: "/api/v1/room-mappings", statusCode: http.StatusAccepted, contentType: "application/json", contains: []string{`"status":"accepted"`}},
		{method: http.MethodPost, path: "/api/v1/annual-reset", statusCode: http.StatusAccepted, contentType: "application/json", contains: []string{`"status":"accepted"`, `"workflow_type":"annual_reset_archive"`}},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.statusCode {
			t.Fatalf("%s %s returned %d, want %d", tc.method, tc.path, rec.Code, tc.statusCode)
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, tc.contentType) {
			t.Fatalf("%s %s content type = %q, want %q", tc.method, tc.path, got, tc.contentType)
		}
		body := rec.Body.String()
		for _, fragment := range tc.contains {
			if !strings.Contains(body, fragment) {
				t.Fatalf("%s %s body missing %q; got %q", tc.method, tc.path, fragment, body)
			}
		}
	}
}

// TestSyncDashboardRoutesRejectInvalidMethodsAndPaths exercises and documents internal/web/sync_dashboard_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestSyncDashboardRoutesRejectInvalidMethodsAndPaths(t *testing.T) {
	handler := web.NewAppHandler(web.HealthDependencies{})

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/sync-dashboard"},
		{method: http.MethodPost, path: "/sync-dashboard/mappings"},
		{method: http.MethodPost, path: "/api/v1/sync-status/pending"},
		{method: http.MethodGet, path: "/api/v1/sync-status/staff/12345/override"},
		{method: http.MethodPost, path: "/api/v1/sync-status/staff/12345"},
		{method: http.MethodDelete, path: "/api/v1/room-mappings"},
		{method: http.MethodGet, path: "/api/v1/annual-reset"},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s %s returned %d, want 404", tc.method, tc.path, rec.Code)
		}
	}
}
