package provider

import (
	"context"
	"fmt"
)

type ReadinessMode string

const (
	ReadinessModeMock     ReadinessMode = "mock"
	ReadinessModeReadOnly ReadinessMode = "read_only"
)

const (
	ProviderNameZoom   = "zoom"
	ProviderNameGoogle = "google"
	ProviderNameAeries = "aeries"
	ProviderNameSFTP   = "sftp"
)

type ReadinessConfig struct {
	Provider        string
	UseMock         bool
	ReadOnly        bool
	Endpoint        string
	CredentialLabel string
}

type ReadinessResult struct {
	Provider string            `json:"provider"`
	Mode     ReadinessMode     `json:"mode"`
	Status   string            `json:"status"`
	Checks   map[string]string `json:"checks"`
}

type ReadinessProbe func(context.Context, ReadinessConfig) error

type ReadinessClient struct {
	config ReadinessConfig
	probe  ReadinessProbe
}

// NewReadinessClient prepares one provider readiness boundary for startup,
// health checks, and provider contract tests. Callers pass sanitized
// configuration metadata, never credential values; mock clients intentionally
// skip the injected probe so DEV evidence can prove initialization does not
// perform real outbound reads or writes.
func NewReadinessClient(config ReadinessConfig, probe ReadinessProbe) (ReadinessClient, error) {
	if config.Provider == "" {
		return ReadinessClient{}, fmt.Errorf("provider readiness config missing provider")
	}
	if !config.UseMock && !config.ReadOnly {
		return ReadinessClient{}, fmt.Errorf("%s readiness must be mock-backed or read-only", config.Provider)
	}
	return ReadinessClient{config: config, probe: probe}, nil
}

// Provider returns the stable provider key used by readiness JSON, tests, and
// operator documentation. It deliberately returns configuration metadata only
// so diagnostics cannot leak credentials or provider payloads.
func (c ReadinessClient) Provider() string {
	return c.config.Provider
}

// CheckReadiness executes the configured provider readiness path. DEV mock
// clients return deterministic success without calling provider SDKs, network
// probes, or write APIs; non-mock clients require an injected read-only probe
// so staging can validate connectivity with environment-owned credentials
// without giving this package direct access to secret material.
func (c ReadinessClient) CheckReadiness(ctx context.Context) (ReadinessResult, error) {
	if c.config.UseMock {
		return ReadinessResult{
			Provider: c.config.Provider,
			Mode:     ReadinessModeMock,
			Status:   "ok",
			Checks: map[string]string{
				"client":         "initialized",
				"outbound_probe": "skipped_mock",
				"writeback":      "disabled",
			},
		}, nil
	}

	if c.probe == nil {
		return ReadinessResult{}, fmt.Errorf("%s readiness probe is not configured", c.config.Provider)
	}
	if err := c.probe(ctx, c.config); err != nil {
		return ReadinessResult{}, fmt.Errorf("%s read-only readiness probe failed: %w", c.config.Provider, err)
	}
	return ReadinessResult{
		Provider: c.config.Provider,
		Mode:     ReadinessModeReadOnly,
		Status:   "ok",
		Checks: map[string]string{
			"client":          "initialized",
			"read_only_probe": "ok",
			"writeback":       "disabled",
		},
	}, nil
}

// CheckReadinessClients runs a group of initialized clients for Phase 0
// evidence and future health wiring. It stops on the first failing provider so
// startup and readiness callers fail closed instead of reporting partial
// success for an ambiguous provider set.
func CheckReadinessClients(ctx context.Context, clients []ReadinessClient) ([]ReadinessResult, error) {
	results := make([]ReadinessResult, 0, len(clients))
	for _, client := range clients {
		result, err := client.CheckReadiness(ctx)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}
