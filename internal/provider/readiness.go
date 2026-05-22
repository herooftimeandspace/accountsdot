package provider

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"strings"
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
	Provider                  string
	UseMock                   bool
	ReadOnly                  bool
	Endpoint                  string
	EndpointEnv               string
	CredentialLabel           string
	CredentialLabelEnv        string
	DatabaseYearMode          string
	MaskedPreviousYearOnly    bool
	CertificateFileConfigured bool
	CertificateFileEnv        string
	CertificateFilePath       string
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

// ConfigurationDiagnostics validates provider metadata before any live SDK
// client or read-only probe is constructed. It returns provider-keyed statuses
// for /health/ready: "mocked" and "ok" are safe states, while "blocked:"
// diagnostics name the fixable environment label without echoing credential,
// URL, private-key, certificate, or service-account contents.
func ConfigurationDiagnostics(configs []ReadinessConfig) map[string]string {
	diagnostics := make(map[string]string, len(configs))
	for _, cfg := range configs {
		diagnostics[cfg.Provider] = cfg.ConfigurationStatus()
	}
	return diagnostics
}

// ConfigurationStatus performs one provider's fail-closed Phase 0 setup
// validation. Mocked providers are ready by definition; live read-only
// providers must have their required non-secret credential label, HTTPS
// endpoint, and certificate file shape before readiness can report ok.
func (cfg ReadinessConfig) ConfigurationStatus() string {
	if cfg.UseMock {
		return "mocked"
	}
	if cfg.Provider == "" {
		return "blocked: missing provider readiness config"
	}
	if !cfg.ReadOnly {
		return fmt.Sprintf("blocked: %s readiness must be read-only", cfg.Provider)
	}
	if cfg.CredentialLabelEnv != "" && strings.TrimSpace(cfg.CredentialLabel) == "" {
		return fmt.Sprintf("blocked: missing required provider setting %s", cfg.CredentialLabelEnv)
	}
	if cfg.EndpointEnv != "" {
		if strings.TrimSpace(cfg.Endpoint) == "" {
			return fmt.Sprintf("blocked: missing required provider setting %s", cfg.EndpointEnv)
		}
		if err := validateReadinessHTTPSURL(cfg.Endpoint); err != nil {
			return fmt.Sprintf("blocked: %s %v", cfg.EndpointEnv, err)
		}
	}
	if cfg.CertificateFileEnv != "" {
		if strings.TrimSpace(cfg.CertificateFilePath) == "" {
			return fmt.Sprintf("blocked: missing required provider setting %s", cfg.CertificateFileEnv)
		}
		if err := validateReadinessCertificateFile(cfg.CertificateFilePath); err != nil {
			return fmt.Sprintf("blocked: %s %v", cfg.CertificateFileEnv, err)
		}
	}
	return "ok"
}

// ConfigurationStatusReady is the health-facing interpretation of provider
// configuration diagnostics. Unknown strings fail closed so new provider states
// cannot accidentally make /health/ready report ok before docs and tests define
// what that state means.
func ConfigurationStatusReady(status string) bool {
	return status == "ok" || status == "mocked"
}

// validateReadinessHTTPSURL checks the shape of a live provider endpoint
// without returning the configured value. This keeps malformed staging URLs
// actionable while preventing hostnames or query fragments from leaking into
// health JSON, logs, issues, or PR evidence.
func validateReadinessHTTPSURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("must be an absolute https URL")
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("must use https")
	}
	return nil
}

// validateReadinessCertificateFile confirms a certificate-backed provider has
// a readable PEM certificate before staging treats live read-only setup as
// ready. It parses only enough structure to prove the file shape and never
// returns the path or certificate material.
func validateReadinessCertificateFile(path string) error {
	data, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return fmt.Errorf("must point to a readable certificate file")
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return fmt.Errorf("must contain a PEM certificate")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return fmt.Errorf("must contain a parseable certificate")
	}
	return nil
}
