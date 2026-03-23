package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/web"
)

func TestAppRoutes(t *testing.T) {
	handler := web.NewAppHandler(web.HealthDependencies{})

	tests := []struct {
		path        string
		contentType string
		contains    string
	}{
		{path: "/", contentType: "text/html", contains: "Go Employee Provisioner"},
		{path: "/metrics", contentType: "text/plain", contains: "app_up 1"},
		{path: "/events/stream", contentType: "text/event-stream", contains: "event: ready"},
		{path: "/api/v1/session/me", contentType: "application/json", contains: `"authenticated":false`},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s returned %d, want 200", tc.path, rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, tc.contentType) {
			t.Fatalf("%s content type = %q, want it to contain %q", tc.path, got, tc.contentType)
		}
		if body := rec.Body.String(); !strings.Contains(body, tc.contains) {
			t.Fatalf("%s body did not contain %q; got %q", tc.path, tc.contains, body)
		}
	}
}

func TestAppRoutesNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()

	web.NewAppHandler(web.HealthDependencies{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
