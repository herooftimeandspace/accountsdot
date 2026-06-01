package provider_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// TestP000D002ProviderReadinessFailureSurfacing verifies the Phase 0 failure
// scenario for provider configuration. The local diagnostics simulate a missing
// credential label, malformed URL, and bad certificate file without contacting
// live providers or echoing configured values into health-ready output.
func TestP000D002ProviderReadinessFailureSurfacing(t *testing.T) {
	tests := []struct {
		name         string
		cfg          provider.ReadinessConfig
		wantContains string
	}{
		{
			name: "missing credential label",
			cfg: provider.ReadinessConfig{
				Provider:           provider.ProviderNameZoom,
				UseMock:            false,
				ReadOnly:           true,
				Endpoint:           "https://zoom.example.test",
				EndpointEnv:        "ZOOM_BASE_URL",
				CredentialLabelEnv: "ZOOM_ACCOUNT_ID",
			},
			wantContains: "ZOOM_ACCOUNT_ID",
		},
		{
			name: "bad url",
			cfg: provider.ReadinessConfig{
				Provider:           provider.ProviderNameZoom,
				UseMock:            false,
				ReadOnly:           true,
				Endpoint:           "http://zoom.example.test",
				EndpointEnv:        "ZOOM_BASE_URL",
				CredentialLabel:    "zoom-staging-label",
				CredentialLabelEnv: "ZOOM_ACCOUNT_ID",
			},
			wantContains: "ZOOM_BASE_URL must use https",
		},
		{
			name: "bad certificate",
			cfg: provider.ReadinessConfig{
				Provider:            provider.ProviderNameAeries,
				UseMock:             false,
				ReadOnly:            true,
				Endpoint:            "https://aeries.example.test",
				EndpointEnv:         "AERIES_BASE_URL",
				CertificateFileEnv:  "AERIES_CERT_FILE",
				CertificateFilePath: writeTempFile(t, "not a certificate"),
			},
			wantContains: "AERIES_CERT_FILE must contain a PEM certificate",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.cfg.ConfigurationStatus()
			if provider.ConfigurationStatusReady(got) {
				t.Fatalf("expected blocked readiness diagnostic, got %q", got)
			}
			if !strings.Contains(got, tc.wantContains) {
				t.Fatalf("diagnostic %q does not contain %q", got, tc.wantContains)
			}
			for _, forbidden := range []string{"zoom-staging-label", "not a certificate"} {
				if strings.Contains(got, forbidden) {
					t.Fatalf("diagnostic leaked configured value %q: %q", forbidden, got)
				}
			}
		})
	}
}

// TestProviderConfigurationDiagnosticsAcceptsMockAndValidLiveConfig locks down
// the safe states consumed by /health/ready: mocked providers are ready without
// live labels, and live read-only providers are ok only after label, URL, and
// certificate-file validation succeeds.
func TestProviderConfigurationDiagnosticsAcceptsMockAndValidLiveConfig(t *testing.T) {
	diagnostics := provider.ConfigurationDiagnostics([]provider.ReadinessConfig{
		{Provider: provider.ProviderNameZoom, UseMock: true},
		{
			Provider:            provider.ProviderNameAeries,
			UseMock:             false,
			ReadOnly:            true,
			Endpoint:            "https://aeries.example.test",
			EndpointEnv:         "AERIES_BASE_URL",
			CertificateFileEnv:  "AERIES_CERT_FILE",
			CertificateFilePath: writeValidCertificate(t),
		},
	})

	if got := diagnostics[provider.ProviderNameZoom]; got != "mocked" {
		t.Fatalf("mocked zoom diagnostic = %q", got)
	}
	if got := diagnostics[provider.ProviderNameAeries]; got != "ok" {
		t.Fatalf("valid aeries diagnostic = %q", got)
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "readiness-test.pem")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp readiness file: %v", err)
	}
	return path
}

func writeValidCertificate(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate readiness key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "readiness.example.test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create readiness certificate: %v", err)
	}
	path := filepath.Join(t.TempDir(), "readiness-test.pem")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatalf("open readiness certificate file: %v", err)
	}
	defer file.Close()
	if err := pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("write readiness certificate: %v", err)
	}
	return path
}
