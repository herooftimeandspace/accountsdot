package config

import (
	"os"
	"strconv"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/auth"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/provider"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/referenceinputs"
)

type Config struct {
	AppEnv               string
	AppPort              string
	ZoomSLGMaxMembers    int
	ReplayEventLimit     int
	ProductionAuthPolicy auth.Policy
	ProviderReadiness    []provider.ReadinessConfig
}

// Load builds startup configuration for the Go service. The application
// entrypoint and config tests call it before any handler or provider setup; it
// validates the repo-local reference-input baseline, reads environment
// variables, applies safe defaults, and returns parsed production-auth policy
// data without contacting Google or validating secrets.
func Load() (Config, error) {
	if err := referenceinputs.ValidateStartup(); err != nil {
		return Config{}, err
	}
	authPolicy, err := loadProductionAuthPolicy()
	if err != nil {
		return Config{}, err
	}
	return Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		AppPort:              getEnv("APP_PORT", "8080"),
		ZoomSLGMaxMembers:    getEnvInt("ZOOM_SLG_MAX_MEMBERS", 10),
		ReplayEventLimit:     getEnvInt("REPLAY_EVENT_LIMIT", 100),
		ProductionAuthPolicy: authPolicy,
		ProviderReadiness:    loadProviderReadinessConfig(),
	}, nil
}

// loadProductionAuthPolicy parses the checked-in production-auth environment
// contract. It creates only configuration data: future SAML middleware will use
// the SAML endpoints and mapping tables to validate Google assertions and apply
// role/site authorization after the domain gate. AUTH_DENIED_EMAIL_DOMAINS may
// add local blocked domains, but the documented student-domain denial remains
// present even when that deployment variable is customized.
func loadProductionAuthPolicy() (auth.Policy, error) {
	policy := auth.DefaultPolicy()
	policy.AllowedEmailDomains = auth.ParseDomainList(getEnv("AUTH_ALLOWED_EMAIL_DOMAINS", auth.DefaultAllowedEmailDomains))
	policy.DeniedEmailDomains = auth.ParseDomainList(auth.DefaultDeniedEmailDomains + "," + getEnv("AUTH_DENIED_EMAIL_DOMAINS", ""))
	policy.SAML = auth.SAMLConfig{
		EntityID:       getEnv("GOOGLE_SAML_ENTITY_ID", ""),
		ACSURL:         getEnv("GOOGLE_SAML_ACS_URL", ""),
		IDPMetadataURL: getEnv("GOOGLE_SAML_IDP_METADATA_URL", ""),
		IDPSSOURL:      getEnv("GOOGLE_SAML_IDP_SSO_URL", ""),
		IDPCertFile:    getEnv("GOOGLE_SAML_IDP_CERT_FILE", ""),
	}

	var err error
	policy.GroupRoleMappings, err = auth.ParseGroupRoleMappings(getEnv("GOOGLE_AUTH_GROUP_ROLE_MAPPINGS_JSON", ""))
	if err != nil {
		return auth.Policy{}, err
	}
	policy.AttributeRoleMappings, err = auth.ParseAttributeRoleMappings(getEnv("GOOGLE_AUTH_ATTRIBUTE_ROLE_MAPPINGS_JSON", ""))
	if err != nil {
		return auth.Policy{}, err
	}
	policy.SiteScopeMappings, err = auth.ParseSiteScopeMappings(getEnv("GOOGLE_AUTH_SITE_SCOPE_MAPPINGS_JSON", ""))
	if err != nil {
		return auth.Policy{}, err
	}
	return policy, nil
}

// loadProviderReadinessConfig collects the non-secret provider metadata used
// by Phase 0 readiness clients. The returned slice names each configured
// provider, whether startup should use the DEV mock client, and the sanitized
// endpoint or credential label that staging probes can echo in diagnostics
// without exposing tokens, certificates, passwords, private keys, or raw
// service-account JSON. The Aeries entry also carries non-secret
// previous-year-read flags so staging evidence can prove masked read-only
// `DatabaseYear=YYYY` setup before any live client is constructed.
func loadProviderReadinessConfig() []provider.ReadinessConfig {
	return []provider.ReadinessConfig{
		{
			Provider:           provider.ProviderNameZoom,
			UseMock:            getEnvBool("USE_MOCK_ZOOM", true),
			ReadOnly:           true,
			Endpoint:           getEnv("ZOOM_BASE_URL", "https://api.zoom.us/v2"),
			EndpointEnv:        "ZOOM_BASE_URL",
			CredentialLabel:    getEnv("ZOOM_ACCOUNT_ID", ""),
			CredentialLabelEnv: "ZOOM_ACCOUNT_ID",
		},
		{
			Provider:           provider.ProviderNameGoogle,
			UseMock:            getEnvBool("USE_MOCK_GOOGLE", true),
			ReadOnly:           true,
			Endpoint:           getEnv("GOOGLE_REDIRECT_URL", ""),
			EndpointEnv:        "GOOGLE_REDIRECT_URL",
			CredentialLabel:    getEnv("GOOGLE_SAML_ENTITY_ID", ""),
			CredentialLabelEnv: "GOOGLE_SAML_ENTITY_ID",
		},
		{
			Provider:                  provider.ProviderNameAeries,
			UseMock:                   getEnvBool("USE_MOCK_AERIES", true),
			ReadOnly:                  getEnvBool("AERIES_READ_ONLY", true),
			Endpoint:                  getEnv("AERIES_BASE_URL", ""),
			EndpointEnv:               "AERIES_BASE_URL",
			DatabaseYearMode:          getEnv("AERIES_DATABASE_YEAR_MODE", ""),
			MaskedPreviousYearOnly:    getEnvBool("AERIES_MASKED_PREVIOUS_YEAR_ONLY", false),
			CertificateFileConfigured: getEnv("AERIES_CERT_FILE", "") != "",
			CertificateFileEnv:        "AERIES_CERT_FILE",
			CertificateFilePath:       getEnv("AERIES_CERT_FILE", ""),
		},
		{
			Provider:           provider.ProviderNameSFTP,
			UseMock:            getEnvBool("USE_MOCK_SFTP", true),
			ReadOnly:           true,
			Endpoint:           getEnv("SFTP_HOST", ""),
			EndpointEnv:        "SFTP_HOST",
			CredentialLabel:    getEnv("SFTP_USERNAME", ""),
			CredentialLabelEnv: "SFTP_USERNAME",
		},
	}
}

// getEnv reads one environment variable and applies the caller's fallback when
// unset. Config loading uses it for non-secret labels and paths as well as
// secret-bearing settings that are never logged or persisted by this package.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt parses integer environment variables used by startup configuration.
// Invalid values fall back to the documented default so local runs do not fail
// from an unrelated typo in an optional tuning variable.
func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// getEnvBool parses feature-flag style environment variables that control
// mock-backed provider setup. Invalid values fall back to the safer caller
// default, which keeps local development mock-backed unless an operator
// explicitly supplies a valid false value for staging read-only probes.
func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
