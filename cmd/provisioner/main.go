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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var runMain = realMain
var runApp = run
var loadConfig = config.Load
var newServer = func(cfg config.Config) server {
	deps, closeHealth := newHealthDependenciesFromEnv()
	return &stdServer{Server: &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           web.NewAppHandler(deps),
		ReadHeaderTimeout: 5 * time.Second,
	}, closeHealth: closeHealth}
}
var notifyContext = signal.NotifyContext
var logPrintf = log.Printf
var logFatalf = log.Fatalf

var errHealthDependencyUnavailable = errors.New("health dependency unavailable")
var errHealthDatabaseConfigInvalid = errors.New("database health configuration invalid")

type server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
	Address() string
}

type stdServer struct {
	*http.Server
	closeHealth func()
}

// Address returns the configured HTTP listen address for startup logging and
// tests. It reads only the stdlib server field created in newServer and has no
// network, database, or provider side effects.
func (s *stdServer) Address() string {
	return s.Addr
}

// Shutdown stops the HTTP server and releases the optional health-check database
// pool opened by newHealthDependenciesFromEnv. Signal handling in run calls this
// method during graceful shutdown.
func (s *stdServer) Shutdown(ctx context.Context) error {
	err := s.Server.Shutdown(ctx)
	if s.closeHealth != nil {
		s.closeHealth()
	}
	return err
}

// main starts the provisioner process. It delegates through runMain so tests can
// replace the entrypoint without starting a real HTTP listener.
func main() {
	runMain()
}

// realMain is the production entrypoint wrapper. It turns run failures into the
// process-fatal log path while keeping run testable as a normal function.
func realMain() {
	if err := runApp(context.Background()); err != nil {
		logFatalf("%v", err)
	}
}

// run loads configuration, builds the HTTP server, and owns graceful shutdown
// for SIGINT/SIGTERM. It does not perform provider writes; startup health
// dependencies are read-only probes that only affect diagnostic responses.
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

// newHealthDependenciesFromEnv wires Phase 0 health checks from DATABASE_URL
// when a deployment provides one. Invalid or unavailable database configuration
// is preserved as readiness failure evidence instead of preventing the
// diagnostics server from starting. Probe callbacks return bounded errors only;
// /health/ready, /health, and /metrics must never receive raw driver text that
// could include hostnames, SQL, usernames, or credential fragments.
func newHealthDependenciesFromEnv() (web.HealthDependencies, func()) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return web.HealthDependencies{}, nil
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return failedHealthDependencies(errHealthDatabaseConfigInvalid), nil
	}

	deps := web.HealthDependencies{
		DBReady: func(parent context.Context) error {
			ctx, cancel := context.WithTimeout(parent, 2*time.Second)
			defer cancel()
			return sanitizeHealthProbeError(pool.Ping(ctx))
		},
		SequenceReady: func(parent context.Context) error {
			ctx, cancel := context.WithTimeout(parent, 2*time.Second)
			defer cancel()
			var allowed bool
			if err := pool.QueryRow(ctx, `select has_sequence_privilege('global_tick_seq', 'USAGE')`).Scan(&allowed); err != nil {
				return sanitizeHealthProbeError(err)
			}
			if !allowed {
				return errHealthDependencyUnavailable
			}
			return nil
		},
		GlobalPaused: func(parent context.Context) (bool, error) {
			ctx, cancel := context.WithTimeout(parent, 2*time.Second)
			defer cancel()
			var paused bool
			err := pool.QueryRow(ctx, `select enabled from system_controls where control_name = 'global_pause'`).Scan(&paused)
			if errors.Is(err, pgx.ErrNoRows) {
				return false, nil
			}
			return paused, sanitizeHealthProbeError(err)
		},
	}

	return deps, pool.Close
}

// sanitizeHealthProbeError converts database driver, SQL, context, and
// configuration errors into bounded sentinel values before they leave
// cmd/provisioner. The web health layer still maps any error to public
// "unavailable" state, but sanitizing here keeps future callers from
// accidentally logging or serializing raw probe details.
func sanitizeHealthProbeError(err error) error {
	if err == nil {
		return nil
	}
	return errHealthDependencyUnavailable
}

// failedHealthDependencies returns callbacks that surface startup probe
// construction failures through /health/ready and /metrics. The original error
// is intentionally discarded so a malformed DATABASE_URL or future setup probe
// cannot leak raw credential-bearing text through a callback result.
func failedHealthDependencies(err error) web.HealthDependencies {
	sanitizedErr := sanitizeHealthProbeError(err)
	return web.HealthDependencies{
		DBReady: func(context.Context) error {
			return sanitizedErr
		},
		SequenceReady: func(context.Context) error {
			return sanitizedErr
		},
		GlobalPaused: func(context.Context) (bool, error) {
			return false, sanitizedErr
		},
	}
}
