package provider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/provider"
)

// TestP000D001ProviderReadinessMockSuccessPath records the DEV verification
// path for Phase 0 scenario P0-0D-001. It initializes every current provider
// readiness client in mock mode, proves the checks pass, and fails if an
// outbound probe is called because DEV mocks must not reach real providers or
// provider write APIs.
func TestP000D001ProviderReadinessMockSuccessPath(t *testing.T) {
	configs := []provider.ReadinessConfig{
		{Provider: provider.ProviderNameZoom, UseMock: true, ReadOnly: true},
		{Provider: provider.ProviderNameGoogle, UseMock: true, ReadOnly: true},
		{Provider: provider.ProviderNameAeries, UseMock: true, ReadOnly: true},
		{Provider: provider.ProviderNameSFTP, UseMock: true, ReadOnly: true},
	}

	probeCalls := 0
	clients := make([]provider.ReadinessClient, 0, len(configs))
	for _, cfg := range configs {
		client, err := provider.NewReadinessClient(cfg, func(context.Context, provider.ReadinessConfig) error {
			probeCalls++
			return errors.New("mock readiness must not call outbound probe")
		})
		if err != nil {
			t.Fatalf("NewReadinessClient(%s) returned error: %v", cfg.Provider, err)
		}
		clients = append(clients, client)
	}

	results, err := provider.CheckReadinessClients(context.Background(), clients)
	if err != nil {
		t.Fatalf("CheckReadinessClients returned error: %v", err)
	}
	if probeCalls != 0 {
		t.Fatalf("mock readiness called outbound probe %d times", probeCalls)
	}
	if len(results) != len(configs) {
		t.Fatalf("readiness result count = %d, want %d", len(results), len(configs))
	}
	for _, result := range results {
		if result.Mode != provider.ReadinessModeMock {
			t.Fatalf("%s mode = %q, want mock", result.Provider, result.Mode)
		}
		if result.Status != "ok" {
			t.Fatalf("%s status = %q, want ok", result.Provider, result.Status)
		}
		if result.Checks["writeback"] != "disabled" {
			t.Fatalf("%s writeback check = %q, want disabled", result.Provider, result.Checks["writeback"])
		}
		if result.Checks["outbound_probe"] != "skipped_mock" {
			t.Fatalf("%s outbound probe check = %q, want skipped_mock", result.Provider, result.Checks["outbound_probe"])
		}
	}
}

// TestReadOnlyProviderReadinessUsesInjectedProbe verifies the staging-facing
// readiness contract without embedding credentials in tests. The caller owns
// the read-only connectivity probe, and this package reports success only
// after that probe returns cleanly.
func TestReadOnlyProviderReadinessUsesInjectedProbe(t *testing.T) {
	probeCalls := 0
	client, err := provider.NewReadinessClient(provider.ReadinessConfig{
		Provider: provider.ProviderNameAeries,
		UseMock:  false,
		ReadOnly: true,
		Endpoint: "https://aeries.example.test",
	}, func(ctx context.Context, cfg provider.ReadinessConfig) error {
		probeCalls++
		if cfg.Endpoint == "" {
			t.Fatal("probe received empty endpoint")
		}
		return ctx.Err()
	})
	if err != nil {
		t.Fatalf("NewReadinessClient returned error: %v", err)
	}

	result, err := client.CheckReadiness(context.Background())
	if err != nil {
		t.Fatalf("CheckReadiness returned error: %v", err)
	}
	if probeCalls != 1 {
		t.Fatalf("read-only readiness probe calls = %d, want 1", probeCalls)
	}
	if result.Mode != provider.ReadinessModeReadOnly || result.Checks["read_only_probe"] != "ok" {
		t.Fatalf("unexpected read-only result: %#v", result)
	}
}

// TestReadinessClientFailsUnsafeLiveWriteMode keeps Phase 0 provider setup
// from silently constructing a non-mock client that is not explicitly read
// only. Later write-capable workers must add their own what-if and pilot-gated
// contracts instead of reusing readiness as a writeback path.
func TestReadinessClientFailsUnsafeLiveWriteMode(t *testing.T) {
	if _, err := provider.NewReadinessClient(provider.ReadinessConfig{
		Provider: provider.ProviderNameZoom,
		UseMock:  false,
		ReadOnly: false,
	}, nil); err == nil {
		t.Fatal("NewReadinessClient accepted a non-mock non-read-only provider")
	}
}
