package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/config"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/web"
)

var runMain = realMain
var runApp = run
var loadConfig = config.Load
var newServer = func(cfg config.Config) server {
	return &stdServer{Server: &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           web.NewAppHandler(web.HealthDependencies{}),
		ReadHeaderTimeout: 5 * time.Second,
	}}
}
var notifyContext = signal.NotifyContext
var logPrintf = log.Printf
var logFatalf = log.Fatalf

type server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
	Address() string
}

type stdServer struct {
	*http.Server
}

// Address documents the data flow for cmd/provisioner/main.go. The application entrypoint reaches this function during startup; debug it when configuration, server lifecycle, or shutdown behavior is unclear. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (s *stdServer) Address() string {
	return s.Addr
}

// main documents the data flow for cmd/provisioner/main.go. The application entrypoint reaches this function during startup; debug it when configuration, server lifecycle, or shutdown behavior is unclear. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func main() {
	runMain()
}

// realMain documents the data flow for cmd/provisioner/main.go. The application entrypoint reaches this function during startup; debug it when configuration, server lifecycle, or shutdown behavior is unclear. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func realMain() {
	if err := runApp(context.Background()); err != nil {
		logFatalf("%v", err)
	}
}

// run documents the data flow for cmd/provisioner/main.go. The application entrypoint reaches this function during startup; debug it when configuration, server lifecycle, or shutdown behavior is unclear. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func run(baseCtx context.Context) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	server := newServer(cfg)

	ctx, stop := notifyContext(baseCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logPrintf("shutdown error: %v", err)
		}
	}()

	logPrintf("starting provisioner on %s in %s mode", server.Address(), cfg.AppEnv)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}
