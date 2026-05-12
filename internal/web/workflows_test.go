package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/web"
)

// TestWorkflowAndApprovalRoutes exercises and documents internal/web/workflows_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestWorkflowAndApprovalRoutes(t *testing.T) {
	handler := web.NewAppHandler(web.HealthDependencies{})

	tests := []struct {
		method      string
		path        string
		statusCode  int
		contentType string
		contains    string
	}{
		{method: http.MethodGet, path: "/api/v1/workflows", statusCode: http.StatusOK, contentType: "application/json", contains: `"items":[]`},
		{method: http.MethodGet, path: "/api/v1/workflows/wf-123", statusCode: http.StatusOK, contentType: "application/json", contains: `"workflow_run_id":"wf-123"`},
		{method: http.MethodGet, path: "/api/v1/approvals", statusCode: http.StatusOK, contentType: "application/json", contains: `"items":[]`},
		{method: http.MethodPost, path: "/api/v1/approvals/ap-1/approve", statusCode: http.StatusAccepted, contentType: "application/json", contains: `"approval_id":"ap-1"`},
		{method: http.MethodPost, path: "/api/v1/workflows/wf-123/retry", statusCode: http.StatusAccepted, contentType: "application/json", contains: `"workflow_run_id":"wf-123"`},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.statusCode {
			t.Fatalf("%s %s returned %d, want %d", tc.method, tc.path, rec.Code, tc.statusCode)
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, tc.contentType) {
			t.Fatalf("%s %s content type = %q, want it to contain %q", tc.method, tc.path, got, tc.contentType)
		}
		if body := rec.Body.String(); !strings.Contains(body, tc.contains) {
			t.Fatalf("%s %s body did not contain %q; got %q", tc.method, tc.path, tc.contains, body)
		}
	}
}

// TestWorkflowAndApprovalRoutesRejectInvalidMethodsAndPaths exercises and documents internal/web/workflows_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestWorkflowAndApprovalRoutesRejectInvalidMethodsAndPaths(t *testing.T) {
	handler := web.NewAppHandler(web.HealthDependencies{})

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workflows"},
		{method: http.MethodGet, path: "/api/v1/workflows/"},
		{method: http.MethodPost, path: "/api/v1/workflows/wf-123"},
		{method: http.MethodGet, path: "/api/v1/workflows/wf-123/retry"},
		{method: http.MethodPost, path: "/api/v1/approvals"},
		{method: http.MethodGet, path: "/api/v1/approvals/ap-1/approve"},
		{method: http.MethodPost, path: "/api/v1/approvals/ap-1/maybe"},
		{method: http.MethodPost, path: "/api/v1/approvals/ap-1"},
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
