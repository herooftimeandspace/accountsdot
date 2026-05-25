package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/web"
)

// TestAppRoutes verifies the application mux still serves the unauthenticated
// smoke routes used by local startup checks. These routes expose no protected
// provider data and give quick feedback before the React DEV frontend is
// running; an unwired app mux reports not-ready metrics instead of false
// readiness.
func TestAppRoutes(t *testing.T) {
	handler := web.NewAppHandler(web.HealthDependencies{})

	tests := []struct {
		path        string
		contentType string
		contains    string
	}{
		{path: "/", contentType: "text/html", contains: "Go Employee Provisioner"},
		{path: "/metrics", contentType: "text/plain", contains: "app_ready 0"},
		{path: "/events/stream", contentType: "text/event-stream", contains: "event: ready"},
		{path: "/api/v1/openapi.json", contentType: "application/json", contains: `"openapi": "3.1.0"`},
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

// TestAppRoutesNotFound keeps the root handler from accidentally turning every
// unknown path into a successful HTML response.
func TestAppRoutesNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()

	web.NewAppHandler(web.HealthDependencies{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// TestMetricsExposePauseAndDependencyState covers the P0-0E-002 observability
// contract through the full app mux. A paused system and a failed dependency
// both clear app_ready while liveness remains a separate app_up signal.
func TestMetricsExposePauseAndDependencyState(t *testing.T) {
	handler := web.NewAppHandler(web.HealthDependencies{
		DBReady:         func(context.Context) error { return nil },
		SequenceReady:   func(context.Context) error { return nil },
		ImportPathReady: func(context.Context) error { return nil },
		SFTPReady:       func(context.Context) error { return nil },
		GoogleReady:     func(context.Context) error { return errMetric{} },
		GlobalPaused:    func(context.Context) (bool, error) { return true, nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"app_up 1",
		"app_ready 0",
		"app_global_pause 1",
		`app_dependency_ready{name="google"} 0`,
		`app_dependency_ready{name="db"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q; got %s", want, body)
		}
	}
}

// TestMetricsExposeProviderReadinessState verifies that provider diagnostics
// get their own bounded Prometheus gauge. Without this signal app_ready can be
// 0 while every legacy dependency gauge remains 1, leaving operators unable to
// identify provider configuration as the readiness blocker.
func TestMetricsExposeProviderReadinessState(t *testing.T) {
	deps := readyMetricsDeps()
	deps.ProviderReady = func(context.Context) map[string]string {
		return map[string]string{
			"aeries": "mocked",
			"zoom":   "blocked: missing required provider setting ZOOM_ACCOUNT_ID",
		}
	}
	handler := web.NewAppHandler(deps)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"app_ready 0",
		`app_dependency_ready{name="db"} 1`,
		`app_provider_ready{name="aeries"} 1`,
		`app_provider_ready{name="zoom"} 0`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q; got %s", want, body)
		}
	}
}

func readyMetricsDeps() web.HealthDependencies {
	return web.HealthDependencies{
		DBReady:         func(context.Context) error { return nil },
		SequenceReady:   func(context.Context) error { return nil },
		ImportPathReady: func(context.Context) error { return nil },
		SFTPReady:       func(context.Context) error { return nil },
		GoogleReady:     func(context.Context) error { return nil },
		GlobalPaused:    func(context.Context) (bool, error) { return false, nil },
	}
}

type errMetric struct{}

// Error returns a deterministic metrics dependency failure string. The metrics
// endpoint converts the failure to a bounded 0/1 gauge instead of exposing this
// text as a Prometheus label.
func (errMetric) Error() string { return "metrics dependency failed" }
