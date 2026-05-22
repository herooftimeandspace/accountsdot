package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/config"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/provider"
)

// TestMainDelegatesToRunMain exercises and documents cmd/provisioner/main_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestMainDelegatesToRunMain(t *testing.T) {
	called := false
	previous := runMain
	runMain = func() {
		called = true
	}
	defer func() {
		runMain = previous
	}()

	main()

	if !called {
		t.Fatal("expected main to delegate to runMain")
	}
}

// TestStdServerAddress exercises and documents cmd/provisioner/main_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestStdServerAddress(t *testing.T) {
	server := &stdServer{Server: &http.Server{Addr: ":9999"}}
	if got := server.Address(); got != ":9999" {
		t.Fatalf("expected address :9999, got %q", got)
	}
}

// TestNewServerReportsInvalidHealthDatabaseConfig verifies startup diagnostics
// stay available when DATABASE_URL cannot be parsed. The health response must
// expose a sanitized readiness failure instead of echoing the raw URL.
func TestNewServerReportsInvalidHealthDatabaseConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "://user:secret@example.invalid/wizard")

	server := newServer(config.Config{AppPort: "8080"})
	std, ok := server.(*stdServer)
	if !ok {
		t.Fatalf("newServer returned %T, want *stdServer", server)
	}

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	std.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"db":"unavailable"`) {
		t.Fatalf("health body = %s, want sanitized database configuration failure", body)
	}
	if strings.Contains(body, "secret") {
		t.Fatalf("health body leaked raw DATABASE_URL: %s", body)
	}
}

// TestNewServerReportsProviderReadinessFailure verifies cmd/provisioner wires
// config.Load provider metadata into /health/ready. A blocked live-mode
// provider must be visible as provider_<name> diagnostics even when the process
// itself remains reachable through /health/live.
func TestNewServerReportsProviderReadinessFailure(t *testing.T) {
	server := newServer(config.Config{
		AppPort: "8080",
		ProviderReadiness: []provider.ReadinessConfig{
			{
				Provider:           provider.ProviderNameZoom,
				UseMock:            false,
				ReadOnly:           true,
				Endpoint:           "https://zoom.example.test/v2",
				EndpointEnv:        "ZOOM_BASE_URL",
				CredentialLabelEnv: "ZOOM_ACCOUNT_ID",
			},
		},
	})
	std, ok := server.(*stdServer)
	if !ok {
		t.Fatalf("newServer returned %T, want *stdServer", server)
	}

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	std.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"provider_zoom":"blocked: missing required provider setting ZOOM_ACCOUNT_ID"`) {
		t.Fatalf("health body = %s, want provider readiness diagnostic", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec = httptest.NewRecorder()
	std.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "provider_zoom") {
		t.Fatalf("live health = %d %s, want process-only ok", rec.Code, rec.Body.String())
	}
}

// TestHealthProbeErrorsAreBounded locks down the cmd/provisioner side of the
// health contract. Even before internal/web maps failures to public
// "unavailable" states, probe helpers must not return raw driver or URL text
// that could later be logged or serialized by a future caller.
func TestHealthProbeErrorsAreBounded(t *testing.T) {
	rawErr := errors.New("dial tcp db.internal.example:5432 password=secret")

	if got := sanitizeHealthProbeError(rawErr); got == nil || strings.Contains(got.Error(), "secret") || strings.Contains(got.Error(), "db.internal") {
		t.Fatalf("sanitizeHealthProbeError(%q) = %v, want bounded non-secret error", rawErr, got)
	}
	if got := sanitizeHealthProbeError(nil); got != nil {
		t.Fatalf("sanitizeHealthProbeError(nil) = %v, want nil", got)
	}

	deps := failedHealthDependencies(rawErr)
	for name, check := range map[string]func(context.Context) error{
		"db":       deps.DBReady,
		"sequence": deps.SequenceReady,
	} {
		err := check(context.Background())
		if err == nil || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "db.internal") {
			t.Fatalf("%s failed health dependency returned %v, want bounded non-secret error", name, err)
		}
	}
	_, err := deps.GlobalPaused(context.Background())
	if err == nil || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "db.internal") {
		t.Fatalf("global pause failed health dependency returned %v, want bounded non-secret error", err)
	}
}

// TestRealMainCallsLogFatalfOnError exercises and documents cmd/provisioner/main_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestRealMainCallsLogFatalfOnError(t *testing.T) {
	restore := overrideMainDeps(t)
	defer restore()

	runApp = func(context.Context) error {
		return errors.New("boom")
	}

	var got string
	logFatalf = func(format string, args ...any) {
		got = format
	}

	realMain()

	if got != "%v" {
		t.Fatalf("expected logFatalf to be called with %%v format, got %q", got)
	}
}

// TestRunReturnsConfigError exercises and documents cmd/provisioner/main_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestRunReturnsConfigError(t *testing.T) {
	restore := overrideMainDeps(t)
	defer restore()

	loadConfig = func() (config.Config, error) {
		return config.Config{}, errors.New("boom")
	}

	err := run(context.Background())
	if err == nil || err.Error() != "load config: boom" {
		t.Fatalf("expected wrapped config error, got %v", err)
	}
}

// TestRunReturnsListenError exercises and documents cmd/provisioner/main_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestRunReturnsListenError(t *testing.T) {
	restore := overrideMainDeps(t)
	defer restore()

	loadConfig = func() (config.Config, error) {
		return config.Config{AppEnv: "test", AppPort: "8080"}, nil
	}
	newServer = func(config.Config) server {
		return &fakeServer{listenErr: errors.New("listen failed"), addr: ":8080"}
	}
	notifyContext = func(ctx context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return context.WithCancel(ctx)
	}
	logPrintf = func(string, ...any) {}

	err := run(context.Background())
	if err == nil || err.Error() != "listen and serve: listen failed" {
		t.Fatalf("expected wrapped listen error, got %v", err)
	}
}

// TestRunTreatsServerClosedAsCleanExit exercises and documents cmd/provisioner/main_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestRunTreatsServerClosedAsCleanExit(t *testing.T) {
	restore := overrideMainDeps(t)
	defer restore()

	loadConfig = func() (config.Config, error) {
		return config.Config{AppEnv: "test", AppPort: "8080"}, nil
	}
	newServer = func(config.Config) server {
		return &fakeServer{listenErr: http.ErrServerClosed, addr: ":8080"}
	}
	notifyContext = func(ctx context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return context.WithCancel(ctx)
	}
	logPrintf = func(string, ...any) {}

	if err := run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// TestRunShutsDownWhenContextIsCanceled exercises and documents cmd/provisioner/main_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/planning/external-write-inventory.md.
func TestRunShutsDownWhenContextIsCanceled(t *testing.T) {
	restore := overrideMainDeps(t)
	defer restore()

	loadConfig = func() (config.Config, error) {
		return config.Config{AppEnv: "test", AppPort: "8080"}, nil
	}

	shutdownCalled := make(chan struct{})
	fake := &fakeServer{
		addr: ":8080",
		listenFunc: func() error {
			<-shutdownCalled
			return http.ErrServerClosed
		},
		shutdownFunc: func(context.Context) error {
			select {
			case <-shutdownCalled:
			default:
				close(shutdownCalled)
			}
			return nil
		},
	}
	newServer = func(config.Config) server { return fake }
	notifyContext = func(context.Context, ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx, func() {}
	}
	logPrintf = func(string, ...any) {}

	if err := run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fake.shutdowns != 1 {
		t.Fatalf("expected 1 shutdown call, got %d", fake.shutdowns)
	}
}

// TestRunLogsShutdownError exercises and documents cmd/provisioner/main_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestRunLogsShutdownError(t *testing.T) {
	restore := overrideMainDeps(t)
	defer restore()

	loadConfig = func() (config.Config, error) {
		return config.Config{AppEnv: "test", AppPort: "8080"}, nil
	}

	shutdownCalled := make(chan struct{})
	fake := &fakeServer{
		addr: ":8080",
		listenFunc: func() error {
			<-shutdownCalled
			return http.ErrServerClosed
		},
		shutdownFunc: func(context.Context) error {
			select {
			case <-shutdownCalled:
			default:
				close(shutdownCalled)
			}
			return errors.New("shutdown failed")
		},
	}
	newServer = func(config.Config) server { return fake }
	notifyContext = func(context.Context, ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx, func() {}
	}
	logged := make(chan string, 4)
	logPrintf = func(format string, args ...any) {
		select {
		case logged <- format:
		default:
		}
	}

	if err := run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	for i := 0; i < 4; i++ {
		select {
		case got := <-logged:
			if got == "shutdown error: %v" {
				return
			}
		default:
			t.Fatal("expected shutdown error to be logged")
		}
	}
	t.Fatal("expected shutdown error to be logged")
}

type fakeServer struct {
	addr         string
	listenErr    error
	listenFunc   func() error
	shutdownFunc func(context.Context) error
	shutdowns    int
}

// ListenAndServe lets run tests decide whether startup blocks, fails, or exits
// as http.ErrServerClosed. It records no network state and delegates to
// listenFunc when a test needs to synchronize shutdown.
func (f *fakeServer) ListenAndServe() error {
	if f.listenFunc != nil {
		return f.listenFunc()
	}
	return f.listenErr
}

// Shutdown records graceful-shutdown attempts made by run's signal goroutine.
// Tests inject shutdownFunc to unblock ListenAndServe or return a controlled
// shutdown error for log-path assertions.
func (f *fakeServer) Shutdown(ctx context.Context) error {
	f.shutdowns++
	if f.shutdownFunc != nil {
		return f.shutdownFunc(ctx)
	}
	return nil
}

// Address returns the fake listen address used in startup log assertions. It
// avoids constructing a real stdServer when tests only need the server contract.
func (f *fakeServer) Address() string {
	return f.addr
}

// overrideMainDeps snapshots package-level seams that main and run use for
// configuration, server construction, signal handling, and logging. Each test
// defers the returned restore function so overrides do not leak across cases.
func overrideMainDeps(t *testing.T) func() {
	t.Helper()

	prevLoadConfig := loadConfig
	prevRunApp := runApp
	prevNewServer := newServer
	prevNotifyContext := notifyContext
	prevLogPrintf := logPrintf
	prevLogFatalf := logFatalf

	return func() {
		loadConfig = prevLoadConfig
		runApp = prevRunApp
		newServer = prevNewServer
		notifyContext = prevNotifyContext
		logPrintf = prevLogPrintf
		logFatalf = prevLogFatalf
	}
}
