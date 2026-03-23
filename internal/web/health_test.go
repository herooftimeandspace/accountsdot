package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/web"
)

func TestHealthRoutes(t *testing.T) {
	handler := web.NewHealthHandler(web.HealthDependencies{
		DBReady:         func() error { return nil },
		SequenceReady:   func() error { return nil },
		ImportPathReady: func() error { return nil },
		SFTPReady:       func() error { return nil },
		GoogleReady:     func() error { return nil },
	})

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

func TestHealthReadyFailsDependency(t *testing.T) {
	handler := web.NewHealthHandler(web.HealthDependencies{
		DBReady:         func() error { return nil },
		SequenceReady:   func() error { return nil },
		ImportPathReady: func() error { return nil },
		SFTPReady:       func() error { return nil },
		GoogleReady:     func() error { return errBoom{} },
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHealthReadyAllowsMissingOptionalCheck(t *testing.T) {
	handler := web.NewHealthHandler(web.HealthDependencies{
		DBReady:         func() error { return nil },
		SequenceReady:   func() error { return nil },
		ImportPathReady: func() error { return nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }
