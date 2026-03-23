package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/config"
)

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

func TestStdServerAddress(t *testing.T) {
	server := &stdServer{Server: &http.Server{Addr: ":9999"}}
	if got := server.Address(); got != ":9999" {
		t.Fatalf("expected address :9999, got %q", got)
	}
}

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

func (f *fakeServer) ListenAndServe() error {
	if f.listenFunc != nil {
		return f.listenFunc()
	}
	return f.listenErr
}

func (f *fakeServer) Shutdown(ctx context.Context) error {
	f.shutdowns++
	if f.shutdownFunc != nil {
		return f.shutdownFunc(ctx)
	}
	return nil
}

func (f *fakeServer) Address() string {
	return f.addr
}

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
