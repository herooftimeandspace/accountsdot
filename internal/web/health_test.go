package web_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/web"
)

// TestHealthRoutes locks down the base Phase 0 health route contract: liveness,
// readiness, and the legacy /health alias return JSON when dependencies pass.
// It calls NewHealthHandler directly so regressions can be debugged without the
// larger application mux.
func TestHealthRoutes(t *testing.T) {
	handler := web.NewHealthHandler(readyDeps())

	tests := []struct {
		path       string
		statusCode int
	}{
		{path: "/health/live", statusCode: http.StatusOK},
		{path: "/health/ready", statusCode: http.StatusOK},
		{path: "/health", statusCode: http.StatusOK},
	}
	for _, tc := range tests {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.statusCode {
			t.Fatalf("%s returned %d, want %d", tc.path, rec.Code, tc.statusCode)
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("%s returned invalid JSON: %v", tc.path, err)
		}
		if payload["status"] == "" {
			t.Fatalf("%s returned empty status payload", tc.path)
		}
	}
}

// TestHealthReadyFailsDependency proves /health/ready fails closed when a
// configured dependency callback returns an error. The response remains JSON and
// uses bounded states so operators can see which dependency blocked readiness
// without exposing raw driver or provider text.
func TestHealthReadyFailsDependency(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*web.HealthDependencies)
		dependency string
	}{
		{name: "db", mutate: func(deps *web.HealthDependencies) { deps.DBReady = func(context.Context) error { return errBoom{} } }, dependency: "db"},
		{name: "import path", mutate: func(deps *web.HealthDependencies) {
			deps.ImportPathReady = func(context.Context) error { return errBoom{} }
		}, dependency: "import_path"},
		{name: "sftp", mutate: func(deps *web.HealthDependencies) { deps.SFTPReady = func(context.Context) error { return errBoom{} } }, dependency: "sftp"},
		{name: "google service account", mutate: func(deps *web.HealthDependencies) {
			deps.GoogleReady = func(context.Context) error { return errBoom{} }
		}, dependency: "google"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deps := readyDeps()
			tc.mutate(&deps)
			handler := web.NewHealthHandler(deps)

			rec := requestHealth(t, handler, "/health/ready", http.StatusServiceUnavailable)
			body := rec.Body.String()
			want := `"` + tc.dependency + `":"unavailable"`
			if !strings.Contains(body, `"status":"degraded"`) || !strings.Contains(body, want) || strings.Contains(body, "secret") {
				t.Fatalf("ready dependency failure body = %s, want sanitized degraded %s signal", body, tc.dependency)
			}
		})
	}
}

// TestHealthReadyFailsClosedOnMissingRequiredDependency covers the Phase 0
// degraded drill where required DB, sequence, import-staging storage, or Google
// service-account checks are absent from the application wiring. Readiness must
// return 503 while liveness still returns a meaningful process-level ok.
func TestHealthReadyFailsClosedOnMissingRequiredDependency(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*web.HealthDependencies)
		dependency string
	}{
		{name: "db", mutate: func(deps *web.HealthDependencies) { deps.DBReady = nil }, dependency: "db"},
		{name: "sequence", mutate: func(deps *web.HealthDependencies) { deps.SequenceReady = nil }, dependency: "sequence"},
		{name: "import path", mutate: func(deps *web.HealthDependencies) { deps.ImportPathReady = nil }, dependency: "import_path"},
		{name: "google service account", mutate: func(deps *web.HealthDependencies) { deps.GoogleReady = nil }, dependency: "google"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deps := readyDeps()
			tc.mutate(&deps)
			handler := web.NewHealthHandler(deps)

			readyRec := requestHealth(t, handler, "/health/ready", http.StatusServiceUnavailable)
			body := readyRec.Body.String()
			want := `"` + tc.dependency + `":"missing_required_check"`
			if !strings.Contains(body, `"status":"degraded"`) || !strings.Contains(body, want) {
				t.Fatalf("ready missing dependency body = %s, want %s", body, want)
			}

			liveRec := requestHealth(t, handler, "/health/live", http.StatusOK)
			if body := liveRec.Body.String(); !strings.Contains(body, `"status":"ok"`) || strings.Contains(body, tc.dependency) {
				t.Fatalf("live missing dependency body = %s, want process-only ok", body)
			}
		})
	}
}

// TestHealthReadyAllowsMissingOptionalSFTPCheck keeps local smoke checks usable
// before SFTP integration mode wires a concrete readiness callback. Missing
// optional SFTP is reported as not_configured instead of failing readiness.
func TestHealthReadyAllowsMissingOptionalSFTPCheck(t *testing.T) {
	deps := readyDeps()
	deps.SFTPReady = nil
	handler := web.NewHealthHandler(deps)

	rec := requestHealth(t, handler, "/health/ready", http.StatusOK)
	if body := rec.Body.String(); !strings.Contains(body, `"sftp":"not_configured"`) {
		t.Fatalf("ready missing optional check body = %s, want not_configured signal", body)
	}
}

// TestHealthReadyFailsWhenPaused proves global pause is observable without
// implying the worker system is ready to claim new work. Liveness remains online
// so diagnostics can be reached while readiness returns a paused 503.
func TestHealthReadyFailsWhenPaused(t *testing.T) {
	handler := web.NewHealthHandler(web.HealthDependencies{
		DBReady:         func(context.Context) error { return nil },
		SequenceReady:   func(context.Context) error { return nil },
		ImportPathReady: func(context.Context) error { return nil },
		SFTPReady:       func(context.Context) error { return nil },
		GoogleReady:     func(context.Context) error { return nil },
		GlobalPaused:    func(context.Context) (bool, error) { return true, nil },
	})

	for _, tc := range []struct {
		path       string
		statusCode int
		status     string
	}{
		{path: "/health/live", statusCode: http.StatusOK, status: `"status":"ok"`},
		{path: "/health/ready", statusCode: http.StatusServiceUnavailable, status: `"status":"paused"`},
		{path: "/health", statusCode: http.StatusServiceUnavailable, status: `"status":"paused"`},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.statusCode {
			t.Fatalf("%s returned %d, want %d", tc.path, rec.Code, tc.statusCode)
		}
		body := rec.Body.String()
		if !strings.Contains(body, tc.status) {
			t.Fatalf("%s body = %s, want %s", tc.path, body, tc.status)
		}
		if tc.path == "/health/live" && strings.Contains(body, "global_pause") {
			t.Fatalf("%s body = %s, want liveness without DB-backed controls", tc.path, body)
		}
		if tc.path != "/health/live" && !strings.Contains(body, `"global_pause":"paused"`) {
			t.Fatalf("%s body = %s, want paused control", tc.path, body)
		}
	}
}

// TestHealthReadyPrefersDegradedWhenPausedAndDependencyFails makes dependency
// loss the top-level readiness status even when global pause is active. That
// distinction keeps pause observability from hiding a separate outage.
func TestHealthReadyPrefersDegradedWhenPausedAndDependencyFails(t *testing.T) {
	handler := web.NewHealthHandler(web.HealthDependencies{
		DBReady:         func(context.Context) error { return errBoom{} },
		SequenceReady:   func(context.Context) error { return nil },
		ImportPathReady: func(context.Context) error { return nil },
		SFTPReady:       func(context.Context) error { return nil },
		GoogleReady:     func(context.Context) error { return nil },
		GlobalPaused:    func(context.Context) (bool, error) { return true, nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"status":"degraded"`) || !strings.Contains(body, `"global_pause":"paused"`) {
		t.Fatalf("ready paused dependency failure body = %s, want degraded status and paused control", body)
	}
}

// TestHealthLiveDoesNotCallGlobalPause keeps liveness free of database-backed
// control checks. A DB outage or slow control query should affect readiness and
// metrics, not the process-alive probe.
func TestHealthLiveDoesNotCallGlobalPause(t *testing.T) {
	called := false
	handler := web.NewHealthHandler(web.HealthDependencies{
		GlobalPaused: func(context.Context) (bool, error) {
			called = true
			return false, errBoom{}
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if called {
		t.Fatal("/health/live called GlobalPaused")
	}
	if body := rec.Body.String(); strings.Contains(body, "unavailable") || strings.Contains(body, "global_pause") {
		t.Fatalf("liveness body = %s, want process-only health payload", body)
	}
}

// TestHealthReadyUsesRequestContext proves request cancellation reaches
// dependency callbacks before they create driver-level timeout contexts. This
// keeps abandoned health requests from continuing DB probes needlessly.
func TestHealthReadyUsesRequestContext(t *testing.T) {
	handler := web.NewHealthHandler(web.HealthDependencies{
		DBReady: func(ctx context.Context) error {
			return ctx.Err()
		},
		GlobalPaused: func(context.Context) (bool, error) {
			return false, nil
		},
	})
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil).WithContext(parent)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"db":"unavailable"`) || strings.Contains(body, "context canceled") {
		t.Fatalf("ready canceled body = %s, want sanitized canceled dependency", body)
	}
}

func readyDeps() web.HealthDependencies {
	return web.HealthDependencies{
		DBReady:         func(context.Context) error { return nil },
		SequenceReady:   func(context.Context) error { return nil },
		ImportPathReady: func(context.Context) error { return nil },
		SFTPReady:       func(context.Context) error { return nil },
		GoogleReady:     func(context.Context) error { return nil },
		GlobalPaused:    func(context.Context) (bool, error) { return false, nil },
	}
}

func requestHealth(t *testing.T, handler http.Handler, path string, statusCode int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != statusCode {
		t.Fatalf("%s returned %d, want %d; body %s", path, rec.Code, statusCode, rec.Body.String())
	}
	return rec
}

type errBoom struct{}

// Error returns text that must stay out of public health responses. Tests use it
// to verify dependency and control failures are surfaced through sanitized
// states instead of raw driver or provider messages.
func (errBoom) Error() string { return "secret database failure" }
