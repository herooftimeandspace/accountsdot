package web

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/auth"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const adminAuthSettingsMaxBodyBytes int64 = 64 * 1024

var configuredExternalSources = []externalSourceConfig{
	{ProviderKey: "google", ProviderLabel: "Google Workspace", RequiredFields: []string{"client_email", "credential_reference"}},
	{ProviderKey: "zoom", ProviderLabel: "Zoom", RequiredFields: []string{"account_id", "credential_reference"}},
	{ProviderKey: "aeries", ProviderLabel: "Aeries SIS", RequiredFields: []string{"base_url", "certificate_reference"}},
	{ProviderKey: "sftp", ProviderLabel: "Escape SFTP", RequiredFields: []string{"host", "username", "credential_reference"}},
}

var (
	adminAuthSettingsStoreMu    sync.Mutex
	adminAuthSettingsStore      authSettingsStorage
	adminAuthSettingsStoreError error
)

type externalSourceConfig struct {
	ProviderKey    string
	ProviderLabel  string
	RequiredFields []string
}

type authSettingsStorage interface {
	Snapshot(ctx context.Context) (authSettingsSnapshot, error)
	SaveRoleMapping(ctx context.Context, request authMappingWriteRequest, actorID string) (authMappingRecord, error)
	DeleteRoleMapping(ctx context.Context, id int64, reason string, actorID string) error
	SaveSiteScopeMapping(ctx context.Context, request authMappingWriteRequest, actorID string) (authMappingRecord, error)
	DeleteSiteScopeMapping(ctx context.Context, id int64, reason string, actorID string) error
	SaveCredentials(ctx context.Context, provider string, request providerCredentialWriteRequest, actorID string) (externalSourceRecord, error)
	ToggleSource(ctx context.Context, provider string, request externalSourceToggleRequest, actorID string) (externalSourceRecord, error)
	RecordProviderTest(ctx context.Context, provider string, request providerTestRecordRequest, actorID string) (externalSourceRecord, error)
	StoredCredentialValues(ctx context.Context, provider string) (map[string]string, error)
}

type authSettingsSnapshot struct {
	RoleMappings      []authMappingRecord    `json:"role_mappings"`
	SiteScopeMappings []authMappingRecord    `json:"site_scope_mappings"`
	ExternalSources   []externalSourceRecord `json:"external_sources"`
	AuditEvents       []authSettingsAudit    `json:"audit_events"`
	GeneratedAt       string                 `json:"generated_at"`
}

type authMappingRecord struct {
	ID              int64     `json:"id"`
	SourceType      string    `json:"source_type"`
	SourceValue     string    `json:"source_value"`
	AttributeValues []string  `json:"attribute_values,omitempty"`
	RoleKeys        []string  `json:"role_keys,omitempty"`
	SiteCodes       []string  `json:"site_codes,omitempty"`
	ActorID         string    `json:"actor_id"`
	Reason          string    `json:"reason"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type externalSourceRecord struct {
	ProviderKey     string                     `json:"provider_key"`
	ProviderLabel   string                     `json:"provider_label"`
	SyncEnabled     bool                       `json:"sync_enabled"`
	LastTestStatus  string                     `json:"last_test_status,omitempty"`
	LastTestSummary string                     `json:"last_test_summary,omitempty"`
	LastTestAt      *time.Time                 `json:"last_test_at,omitempty"`
	Credentials     []providerCredentialRecord `json:"credentials"`
	ActorID         string                     `json:"actor_id"`
	Reason          string                     `json:"reason"`
	UpdatedAt       time.Time                  `json:"updated_at"`
}

type providerCredentialRecord struct {
	FieldKey    string    `json:"field_key"`
	Stored      bool      `json:"stored"`
	KeyID       string    `json:"key_id"`
	Fingerprint string    `json:"fingerprint"`
	Label       string    `json:"label"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type authSettingsAudit struct {
	ID           int64           `json:"id"`
	ActorID      string          `json:"actor_id"`
	ActorType    string          `json:"actor_type"`
	TargetEntity string          `json:"target_entity"`
	TargetID     string          `json:"target_id"`
	Reason       string          `json:"reason"`
	Diff         json.RawMessage `json:"diff"`
	CreatedAt    time.Time       `json:"created_at"`
}

type authMappingWriteRequest struct {
	SourceType      string   `json:"source_type"`
	SourceValue     string   `json:"source_value"`
	AttributeValues []string `json:"attribute_values"`
	RoleKeys        []string `json:"role_keys"`
	SiteCodes       []string `json:"site_codes"`
	Reason          string   `json:"reason"`
}

type authMappingDeleteRequest struct {
	Reason string `json:"reason"`
}

type authPreviewRequest struct {
	Email      string              `json:"email"`
	Groups     []string            `json:"groups"`
	OUs        []string            `json:"ous"`
	Attributes map[string][]string `json:"attributes"`
}

type providerCredentialWriteRequest struct {
	Fields map[string]string `json:"fields"`
	Labels map[string]string `json:"labels"`
	Reason string            `json:"reason"`
}

type externalSourceToggleRequest struct {
	SyncEnabled bool   `json:"sync_enabled"`
	Reason      string `json:"reason"`
}

type providerTestRequest struct {
	Reason string `json:"reason"`
}

type providerTestRecordRequest struct {
	Status  string
	Summary string
	Reason  string
}

type memoryAuthSettingsStore struct {
	mu          sync.Mutex
	nextID      int64
	nextAuditID int64
	roles       []authMappingRecord
	scopes      []authMappingRecord
	sources     map[string]externalSourceRecord
	secrets     map[string]map[string]string
	audit       []authSettingsAudit
}

type postgresAuthSettingsStore struct {
	pool *pgxpool.Pool
}

func registerAdminAuthSettingsRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/admin/auth-settings", http.HandlerFunc(handleAdminAuthSettings))
	mux.Handle("/api/v1/admin/auth-settings/", http.HandlerFunc(handleAdminAuthSettings))
	mux.Handle("/api/v1/admin/external-sources/", http.HandlerFunc(handleAdminExternalSources))
}

// ResetAdminAuthSettingsForTest clears the memoized admin-auth-settings store so
// tests can isolate in-memory state after changing DATABASE_URL or ENCRYPTION_KEY.
func ResetAdminAuthSettingsForTest() {
	adminAuthSettingsStoreMu.Lock()
	defer adminAuthSettingsStoreMu.Unlock()
	adminAuthSettingsStore = nil
	adminAuthSettingsStoreError = nil
}

func handleAdminAuthSettings(w http.ResponseWriter, r *http.Request) {
	config, ok := requireITAdminAuthSettingsAccess(w, r, "Auth Settings are available to IT Admin only.")
	if !ok {
		return
	}
	store, err := currentAuthSettingsStore(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "storage_unavailable", "message": "Auth settings storage is unavailable."})
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/admin/auth-settings"), "/")
	switch {
	case path == "" && r.Method == http.MethodGet:
		snapshot, err := store.Snapshot(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "snapshot_failed", "message": "Could not load auth settings."})
			return
		}
		writeJSON(w, http.StatusOK, snapshot)
	case path == "role-mappings" && r.Method == http.MethodPost:
		request, err := decodeAuthMappingWriteRequest(w, r, true)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_request", "message": err.Error()})
			return
		}
		record, err := store.SaveRoleMapping(r.Context(), request, config.Persona.ID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "save_failed", "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, record)
	case strings.HasPrefix(path, "role-mappings/") && r.Method == http.MethodDelete:
		id, err := strconv.ParseInt(strings.TrimPrefix(path, "role-mappings/"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_id", "message": "Mapping id must be numeric."})
			return
		}
		request, err := decodeMappingDeleteRequest(w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_request", "message": err.Error()})
			return
		}
		if err := store.DeleteRoleMapping(r.Context(), id, request.Reason, config.Persona.ID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "delete_failed", "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	case path == "site-scope-mappings" && r.Method == http.MethodPost:
		request, err := decodeAuthMappingWriteRequest(w, r, false)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_request", "message": err.Error()})
			return
		}
		record, err := store.SaveSiteScopeMapping(r.Context(), request, config.Persona.ID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "save_failed", "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, record)
	case strings.HasPrefix(path, "site-scope-mappings/") && r.Method == http.MethodDelete:
		id, err := strconv.ParseInt(strings.TrimPrefix(path, "site-scope-mappings/"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_id", "message": "Mapping id must be numeric."})
			return
		}
		request, err := decodeMappingDeleteRequest(w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_request", "message": err.Error()})
			return
		}
		if err := store.DeleteSiteScopeMapping(r.Context(), id, request.Reason, config.Persona.ID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "delete_failed", "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	case path == "preview" && r.Method == http.MethodPost:
		request, err := decodeAuthPreviewRequest(w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_request", "message": err.Error()})
			return
		}
		snapshot, err := store.Snapshot(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "snapshot_failed", "message": "Could not load mappings for preview."})
			return
		}
		writeJSON(w, http.StatusOK, buildAuthPreview(snapshot, request))
	default:
		http.NotFound(w, r)
	}
}

func handleAdminExternalSources(w http.ResponseWriter, r *http.Request) {
	config, ok := requireITAdminAuthSettingsAccess(w, r, "External source settings are available to IT Admin only.")
	if !ok {
		return
	}
	store, err := currentAuthSettingsStore(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "storage_unavailable", "message": "External source storage is unavailable."})
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/admin/external-sources/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" || !configuredProvider(parts[0]) {
		http.NotFound(w, r)
		return
	}
	provider := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	switch {
	case action == "credentials" && r.Method == http.MethodPut:
		request, err := decodeProviderCredentialWriteRequest(w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_request", "message": err.Error()})
			return
		}
		record, err := store.SaveCredentials(r.Context(), provider, request, config.Persona.ID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "credential_save_failed", "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, record)
	case action == "" && r.Method == http.MethodPatch:
		request, err := decodeExternalSourceToggleRequest(w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_request", "message": err.Error()})
			return
		}
		record, err := store.ToggleSource(r.Context(), provider, request, config.Persona.ID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "toggle_failed", "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, record)
	case action == "test" && r.Method == http.MethodPost:
		request, err := decodeProviderTestRequest(w, r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_request", "message": err.Error()})
			return
		}
		values, err := store.StoredCredentialValues(r.Context(), provider)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "credential_read_failed", "message": "Saved credentials could not be read."})
			return
		}
		status, summary := validateProviderCredentials(provider, values)
		record, err := store.RecordProviderTest(r.Context(), provider, providerTestRecordRequest{Status: status, Summary: summary, Reason: request.Reason}, config.Persona.ID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "test_record_failed", "message": err.Error()})
			return
		}
		code := http.StatusOK
		if status != "passed" {
			code = http.StatusBadRequest
		}
		writeJSON(w, code, map[string]any{
			"provider_key": provider,
			"status":       status,
			"summary":      summary,
			"source":       record,
		})
	default:
		http.NotFound(w, r)
	}
}

func requireITAdminAuthSettingsAccess(w http.ResponseWriter, r *http.Request, forbiddenMessage string) (devPersonaConfig, bool) {
	if !devSessionConsumerEnabled(r) {
		http.NotFound(w, r)
		return devPersonaConfig{}, false
	}
	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "not_authorized", "message": "You need to sign in before you can manage Auth Settings."})
		return devPersonaConfig{}, false
	}
	if config.Persona.ID != "it_admin" {
		writeJSON(w, http.StatusForbidden, map[string]any{"code": "forbidden", "message": forbiddenMessage, "persona": config.Persona})
		return devPersonaConfig{}, false
	}
	return config, true
}

func currentAuthSettingsStore(ctx context.Context) (authSettingsStorage, error) {
	adminAuthSettingsStoreMu.Lock()
	defer adminAuthSettingsStoreMu.Unlock()
	if adminAuthSettingsStore != nil || adminAuthSettingsStoreError != nil {
		return adminAuthSettingsStore, adminAuthSettingsStoreError
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		adminAuthSettingsStore = newMemoryAuthSettingsStore()
		return adminAuthSettingsStore, nil
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		adminAuthSettingsStoreError = err
		return nil, err
	}
	adminAuthSettingsStore = postgresAuthSettingsStore{pool: pool}
	return adminAuthSettingsStore, nil
}

func newMemoryAuthSettingsStore() *memoryAuthSettingsStore {
	store := &memoryAuthSettingsStore{
		nextID:      1,
		nextAuditID: 1,
		sources:     make(map[string]externalSourceRecord),
		secrets:     make(map[string]map[string]string),
	}
	for _, source := range configuredExternalSources {
		store.sources[source.ProviderKey] = externalSourceRecord{
			ProviderKey:   source.ProviderKey,
			ProviderLabel: source.ProviderLabel,
			SyncEnabled:   false,
			Credentials:   []providerCredentialRecord{},
			ActorID:       "system",
			Reason:        "registry_default_off",
			UpdatedAt:     time.Now().UTC(),
		}
	}
	return store
}

func (store *memoryAuthSettingsStore) Snapshot(_ context.Context) (authSettingsSnapshot, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return authSettingsSnapshot{
		RoleMappings:      slices.Clone(store.roles),
		SiteScopeMappings: slices.Clone(store.scopes),
		ExternalSources:   store.orderedSourcesLocked(),
		AuditEvents:       slices.Clone(store.audit),
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (store *memoryAuthSettingsStore) SaveRoleMapping(_ context.Context, request authMappingWriteRequest, actorID string) (authMappingRecord, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, err := store.upsertMappingLocked(&store.roles, request, actorID, true)
	if err != nil {
		return authMappingRecord{}, err
	}
	store.appendAuditLocked("auth_role_mapping", strconv.FormatInt(record.ID, 10), request.Reason, actorID, map[string]any{"changed_fields": changedMappingFields(request, true), "redacted": true})
	return record, nil
}

func (store *memoryAuthSettingsStore) DeleteRoleMapping(_ context.Context, id int64, reason string, actorID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if strings.TrimSpace(reason) == "" {
		return errors.New("change reason is required")
	}
	ok := deleteMappingByID(&store.roles, id)
	if !ok {
		return errors.New("role mapping not found")
	}
	store.appendAuditLocked("auth_role_mapping", strconv.FormatInt(id, 10), reason, actorID, map[string]any{"action": "delete", "redacted": true})
	return nil
}

func (store *memoryAuthSettingsStore) SaveSiteScopeMapping(_ context.Context, request authMappingWriteRequest, actorID string) (authMappingRecord, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, err := store.upsertMappingLocked(&store.scopes, request, actorID, false)
	if err != nil {
		return authMappingRecord{}, err
	}
	store.appendAuditLocked("auth_site_scope_mapping", strconv.FormatInt(record.ID, 10), request.Reason, actorID, map[string]any{"changed_fields": changedMappingFields(request, false), "redacted": true})
	return record, nil
}

func (store *memoryAuthSettingsStore) DeleteSiteScopeMapping(_ context.Context, id int64, reason string, actorID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if strings.TrimSpace(reason) == "" {
		return errors.New("change reason is required")
	}
	ok := deleteMappingByID(&store.scopes, id)
	if !ok {
		return errors.New("site-scope mapping not found")
	}
	store.appendAuditLocked("auth_site_scope_mapping", strconv.FormatInt(id, 10), reason, actorID, map[string]any{"action": "delete", "redacted": true})
	return nil
}

func (store *memoryAuthSettingsStore) SaveCredentials(_ context.Context, provider string, request providerCredentialWriteRequest, actorID string) (externalSourceRecord, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if strings.TrimSpace(request.Reason) == "" {
		return externalSourceRecord{}, errors.New("change reason is required")
	}
	source, ok := store.sources[provider]
	if !ok {
		return externalSourceRecord{}, errors.New("provider is not configured")
	}
	if len(request.Fields) == 0 {
		return externalSourceRecord{}, errors.New("at least one credential field is required")
	}
	if store.secrets[provider] == nil {
		store.secrets[provider] = make(map[string]string)
	}
	now := time.Now().UTC()
	for field, value := range request.Fields {
		field = strings.TrimSpace(field)
		if field == "" || strings.TrimSpace(value) == "" {
			return externalSourceRecord{}, errors.New("credential fields and values are required")
		}
		encrypted, keyID, fingerprint, err := encryptCredential(value)
		if err != nil {
			return externalSourceRecord{}, err
		}
		store.secrets[provider][field] = encrypted
		source.Credentials = upsertCredentialMeta(source.Credentials, providerCredentialRecord{
			FieldKey:    field,
			Stored:      true,
			KeyID:       keyID,
			Fingerprint: fingerprint,
			Label:       sanitizedCredentialLabel(request.Labels[field], field),
			UpdatedAt:   now,
		})
	}
	source.ActorID = actorID
	source.Reason = request.Reason
	source.UpdatedAt = now
	store.sources[provider] = source
	store.appendAuditLocked("external_provider_credentials", provider, request.Reason, actorID, map[string]any{"changed_fields": sortedMapKeys(request.Fields), "redacted": true})
	return source, nil
}

func (store *memoryAuthSettingsStore) ToggleSource(_ context.Context, provider string, request externalSourceToggleRequest, actorID string) (externalSourceRecord, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if strings.TrimSpace(request.Reason) == "" {
		return externalSourceRecord{}, errors.New("change reason is required")
	}
	source, ok := store.sources[provider]
	if !ok {
		return externalSourceRecord{}, errors.New("provider is not configured")
	}
	before := source.SyncEnabled
	source.SyncEnabled = request.SyncEnabled
	source.ActorID = actorID
	source.Reason = request.Reason
	source.UpdatedAt = time.Now().UTC()
	store.sources[provider] = source
	store.appendAuditLocked("external_data_source", provider, request.Reason, actorID, map[string]any{"changed_fields": []string{"sync_enabled"}, "before_enabled": before, "after_enabled": request.SyncEnabled, "redacted": true})
	return source, nil
}

func (store *memoryAuthSettingsStore) RecordProviderTest(_ context.Context, provider string, request providerTestRecordRequest, actorID string) (externalSourceRecord, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if strings.TrimSpace(request.Reason) == "" {
		return externalSourceRecord{}, errors.New("change reason is required")
	}
	source, ok := store.sources[provider]
	if !ok {
		return externalSourceRecord{}, errors.New("provider is not configured")
	}
	now := time.Now().UTC()
	source.LastTestStatus = request.Status
	source.LastTestSummary = request.Summary
	source.LastTestAt = &now
	source.ActorID = actorID
	source.Reason = request.Reason
	source.UpdatedAt = now
	store.sources[provider] = source
	store.appendAuditLocked("external_data_source_test", provider, request.Reason, actorID, map[string]any{"status": request.Status, "provider_payload_stored": false, "redacted": true})
	return source, nil
}

func (store *memoryAuthSettingsStore) StoredCredentialValues(_ context.Context, provider string) (map[string]string, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	encryptedValues := store.secrets[provider]
	values := make(map[string]string, len(encryptedValues))
	for field, encrypted := range encryptedValues {
		value, err := decryptCredential(encrypted)
		if err != nil {
			return nil, err
		}
		values[field] = value
	}
	return values, nil
}

func (store *memoryAuthSettingsStore) upsertMappingLocked(records *[]authMappingRecord, request authMappingWriteRequest, actorID string, roles bool) (authMappingRecord, error) {
	if err := validateMappingRequest(request, roles); err != nil {
		return authMappingRecord{}, err
	}
	now := time.Now().UTC()
	for index, record := range *records {
		if record.SourceType == request.SourceType && record.SourceValue == request.SourceValue {
			record.AttributeValues = sanitizeList(request.AttributeValues)
			record.RoleKeys = sanitizeList(request.RoleKeys)
			record.SiteCodes = sanitizeList(request.SiteCodes)
			record.ActorID = actorID
			record.Reason = request.Reason
			record.UpdatedAt = now
			(*records)[index] = record
			return record, nil
		}
	}
	record := authMappingRecord{
		ID:              store.nextID,
		SourceType:      request.SourceType,
		SourceValue:     request.SourceValue,
		AttributeValues: sanitizeList(request.AttributeValues),
		RoleKeys:        sanitizeList(request.RoleKeys),
		SiteCodes:       sanitizeList(request.SiteCodes),
		ActorID:         actorID,
		Reason:          request.Reason,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.nextID++
	*records = append(*records, record)
	return record, nil
}

func (store *memoryAuthSettingsStore) appendAuditLocked(entity string, targetID string, reason string, actorID string, diff map[string]any) {
	rawDiff, _ := json.Marshal(diff)
	store.audit = append([]authSettingsAudit{{
		ID:           store.nextAuditID,
		ActorID:      actorID,
		ActorType:    "dev_persona",
		TargetEntity: entity,
		TargetID:     targetID,
		Reason:       reason,
		Diff:         rawDiff,
		CreatedAt:    time.Now().UTC(),
	}}, store.audit...)
	store.nextAuditID++
}

func (store *memoryAuthSettingsStore) orderedSourcesLocked() []externalSourceRecord {
	sources := make([]externalSourceRecord, 0, len(configuredExternalSources))
	for _, config := range configuredExternalSources {
		source := store.sources[config.ProviderKey]
		source.Credentials = slices.Clone(source.Credentials)
		sources = append(sources, source)
	}
	return sources
}

func (store postgresAuthSettingsStore) Snapshot(ctx context.Context) (authSettingsSnapshot, error) {
	var snapshot authSettingsSnapshot
	err := db.WithRetry(ctx, store.pool, func(tx pgx.Tx) error {
		if err := ensureAuthSettingsRegistry(ctx, tx); err != nil {
			return err
		}
		var err error
		snapshot.RoleMappings, err = queryAuthMappings(ctx, tx, "auth_role_mappings", "role_keys")
		if err != nil {
			return err
		}
		snapshot.SiteScopeMappings, err = queryAuthMappings(ctx, tx, "auth_site_scope_mappings", "site_codes")
		if err != nil {
			return err
		}
		snapshot.ExternalSources, err = queryExternalSources(ctx, tx)
		if err != nil {
			return err
		}
		snapshot.AuditEvents, err = queryAuthSettingsAudit(ctx, tx)
		return err
	})
	snapshot.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	return snapshot, err
}

func (store postgresAuthSettingsStore) SaveRoleMapping(ctx context.Context, request authMappingWriteRequest, actorID string) (authMappingRecord, error) {
	return store.saveMapping(ctx, "auth_role_mappings", "role_keys", "auth_role_mapping", request, actorID, true)
}

func (store postgresAuthSettingsStore) DeleteRoleMapping(ctx context.Context, id int64, reason string, actorID string) error {
	return store.deleteMapping(ctx, "auth_role_mappings", "auth_role_mapping", id, reason, actorID)
}

func (store postgresAuthSettingsStore) SaveSiteScopeMapping(ctx context.Context, request authMappingWriteRequest, actorID string) (authMappingRecord, error) {
	return store.saveMapping(ctx, "auth_site_scope_mappings", "site_codes", "auth_site_scope_mapping", request, actorID, false)
}

func (store postgresAuthSettingsStore) DeleteSiteScopeMapping(ctx context.Context, id int64, reason string, actorID string) error {
	return store.deleteMapping(ctx, "auth_site_scope_mappings", "auth_site_scope_mapping", id, reason, actorID)
}

func (store postgresAuthSettingsStore) SaveCredentials(ctx context.Context, provider string, request providerCredentialWriteRequest, actorID string) (externalSourceRecord, error) {
	if !configuredProvider(provider) {
		return externalSourceRecord{}, errors.New("provider is not configured")
	}
	if strings.TrimSpace(request.Reason) == "" {
		return externalSourceRecord{}, errors.New("change reason is required")
	}
	if len(request.Fields) == 0 {
		return externalSourceRecord{}, errors.New("at least one credential field is required")
	}
	var record externalSourceRecord
	err := db.WithRetry(ctx, store.pool, func(tx pgx.Tx) error {
		if err := ensureAuthSettingsRegistry(ctx, tx); err != nil {
			return err
		}
		now := time.Now().UTC()
		for field, value := range request.Fields {
			field = strings.TrimSpace(field)
			if field == "" || strings.TrimSpace(value) == "" {
				return errors.New("credential fields and values are required")
			}
			encrypted, keyID, fingerprint, err := encryptCredential(value)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				insert into external_provider_credentials (provider_key, field_key, encrypted_value, key_id, fingerprint, label, actor_id, reason, created_at, updated_at)
				values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
				on conflict (provider_key, field_key) do update
				set encrypted_value = excluded.encrypted_value,
					key_id = excluded.key_id,
					fingerprint = excluded.fingerprint,
					label = excluded.label,
					actor_id = excluded.actor_id,
					reason = excluded.reason,
					updated_at = excluded.updated_at
			`, provider, field, encrypted, keyID, fingerprint, sanitizedCredentialLabel(request.Labels[field], field), actorID, request.Reason, now); err != nil {
				return err
			}
		}
		if err := insertAuthSettingsAudit(ctx, tx, actorID, "external_provider_credentials", provider, request.Reason, map[string]any{"changed_fields": sortedMapKeys(request.Fields), "redacted": true}); err != nil {
			return err
		}
		var err error
		record, err = queryExternalSource(ctx, tx, provider)
		return err
	})
	return record, err
}

func (store postgresAuthSettingsStore) ToggleSource(ctx context.Context, provider string, request externalSourceToggleRequest, actorID string) (externalSourceRecord, error) {
	if !configuredProvider(provider) {
		return externalSourceRecord{}, errors.New("provider is not configured")
	}
	if strings.TrimSpace(request.Reason) == "" {
		return externalSourceRecord{}, errors.New("change reason is required")
	}
	var record externalSourceRecord
	err := db.WithRetry(ctx, store.pool, func(tx pgx.Tx) error {
		if err := ensureAuthSettingsRegistry(ctx, tx); err != nil {
			return err
		}
		var before bool
		if err := tx.QueryRow(ctx, `select sync_enabled from external_data_sources where provider_key = $1`, provider).Scan(&before); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			update external_data_sources
			set sync_enabled = $2, actor_id = $3, reason = $4, updated_at = now()
			where provider_key = $1
		`, provider, request.SyncEnabled, actorID, request.Reason); err != nil {
			return err
		}
		if err := insertAuthSettingsAudit(ctx, tx, actorID, "external_data_source", provider, request.Reason, map[string]any{"changed_fields": []string{"sync_enabled"}, "before_enabled": before, "after_enabled": request.SyncEnabled, "redacted": true}); err != nil {
			return err
		}
		var err error
		record, err = queryExternalSource(ctx, tx, provider)
		return err
	})
	return record, err
}

func (store postgresAuthSettingsStore) RecordProviderTest(ctx context.Context, provider string, request providerTestRecordRequest, actorID string) (externalSourceRecord, error) {
	if !configuredProvider(provider) {
		return externalSourceRecord{}, errors.New("provider is not configured")
	}
	if strings.TrimSpace(request.Reason) == "" {
		return externalSourceRecord{}, errors.New("change reason is required")
	}
	var record externalSourceRecord
	err := db.WithRetry(ctx, store.pool, func(tx pgx.Tx) error {
		if err := ensureAuthSettingsRegistry(ctx, tx); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			update external_data_sources
			set last_test_status = $2,
				last_test_summary = $3,
				last_test_at = now(),
				actor_id = $4,
				reason = $5,
				updated_at = now()
			where provider_key = $1
		`, provider, request.Status, request.Summary, actorID, request.Reason); err != nil {
			return err
		}
		if err := insertAuthSettingsAudit(ctx, tx, actorID, "external_data_source_test", provider, request.Reason, map[string]any{"status": request.Status, "provider_payload_stored": false, "redacted": true}); err != nil {
			return err
		}
		var err error
		record, err = queryExternalSource(ctx, tx, provider)
		return err
	})
	return record, err
}

func (store postgresAuthSettingsStore) StoredCredentialValues(ctx context.Context, provider string) (map[string]string, error) {
	values := make(map[string]string)
	err := db.WithRetry(ctx, store.pool, func(tx pgx.Tx) error {
		if err := ensureAuthSettingsRegistry(ctx, tx); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `select field_key, encrypted_value from external_provider_credentials where provider_key = $1`, provider)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var field, encrypted string
			if err := rows.Scan(&field, &encrypted); err != nil {
				return err
			}
			value, err := decryptCredential(encrypted)
			if err != nil {
				return err
			}
			values[field] = value
		}
		return rows.Err()
	})
	return values, err
}

func (store postgresAuthSettingsStore) saveMapping(ctx context.Context, table string, valueColumn string, auditEntity string, request authMappingWriteRequest, actorID string, roles bool) (authMappingRecord, error) {
	if err := validateMappingRequest(request, roles); err != nil {
		return authMappingRecord{}, err
	}
	var record authMappingRecord
	err := db.WithRetry(ctx, store.pool, func(tx pgx.Tx) error {
		attrJSON, err := json.Marshal(sanitizeList(request.AttributeValues))
		if err != nil {
			return err
		}
		values := sanitizeList(request.SiteCodes)
		if roles {
			values = sanitizeList(request.RoleKeys)
		}
		valueJSON, err := json.Marshal(values)
		if err != nil {
			return err
		}
		query := fmt.Sprintf(`
			insert into %s (source_type, source_value, attribute_values, %s, actor_id, reason, created_at, updated_at)
			values ($1, $2, $3::jsonb, $4::jsonb, $5, $6, now(), now())
			on conflict (source_type, source_value) do update
			set attribute_values = excluded.attribute_values,
				%s = excluded.%s,
				actor_id = excluded.actor_id,
				reason = excluded.reason,
				updated_at = now()
			returning id, source_type, source_value, attribute_values, %s, actor_id, reason, created_at, updated_at
		`, table, valueColumn, valueColumn, valueColumn, valueColumn)
		var attrRaw []byte
		var valueRaw []byte
		if err := tx.QueryRow(ctx, query, request.SourceType, request.SourceValue, string(attrJSON), string(valueJSON), actorID, request.Reason).Scan(
			&record.ID,
			&record.SourceType,
			&record.SourceValue,
			&attrRaw,
			&valueRaw,
			&record.ActorID,
			&record.Reason,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return err
		}
		if err := json.Unmarshal(attrRaw, &record.AttributeValues); err != nil {
			return err
		}
		if err := json.Unmarshal(valueRaw, &values); err != nil {
			return err
		}
		if roles {
			record.RoleKeys = values
		} else {
			record.SiteCodes = values
		}
		return insertAuthSettingsAudit(ctx, tx, actorID, auditEntity, strconv.FormatInt(record.ID, 10), request.Reason, map[string]any{"changed_fields": changedMappingFields(request, roles), "redacted": true})
	})
	return record, err
}

func (store postgresAuthSettingsStore) deleteMapping(ctx context.Context, table string, auditEntity string, id int64, reason string, actorID string) error {
	if strings.TrimSpace(reason) == "" {
		return errors.New("change reason is required")
	}
	return db.WithRetry(ctx, store.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, fmt.Sprintf(`delete from %s where id = $1`, table), id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("mapping not found")
		}
		return insertAuthSettingsAudit(ctx, tx, actorID, auditEntity, strconv.FormatInt(id, 10), reason, map[string]any{"action": "delete", "redacted": true})
	})
}

func ensureAuthSettingsRegistry(ctx context.Context, tx pgx.Tx) error {
	for _, source := range configuredExternalSources {
		if _, err := tx.Exec(ctx, `
			insert into external_data_sources (provider_key, provider_label, sync_enabled, actor_id, reason, created_at, updated_at)
			values ($1, $2, false, 'registry', 'registry_default_off', now(), now())
			on conflict (provider_key) do update
			set provider_label = excluded.provider_label
			where external_data_sources.provider_label is distinct from excluded.provider_label
		`, source.ProviderKey, source.ProviderLabel); err != nil {
			return err
		}
	}
	return nil
}

func queryAuthMappings(ctx context.Context, tx pgx.Tx, table string, valueColumn string) ([]authMappingRecord, error) {
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		select id, source_type, source_value, attribute_values, %s, actor_id, reason, created_at, updated_at
		from %s
		order by source_type, source_value
	`, valueColumn, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []authMappingRecord
	for rows.Next() {
		var record authMappingRecord
		var values []string
		var attrRaw []byte
		var valueRaw []byte
		if err := rows.Scan(&record.ID, &record.SourceType, &record.SourceValue, &attrRaw, &valueRaw, &record.ActorID, &record.Reason, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(attrRaw, &record.AttributeValues); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(valueRaw, &values); err != nil {
			return nil, err
		}
		if valueColumn == "role_keys" {
			record.RoleKeys = values
		} else {
			record.SiteCodes = values
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func queryExternalSources(ctx context.Context, tx pgx.Tx) ([]externalSourceRecord, error) {
	sources := make([]externalSourceRecord, 0, len(configuredExternalSources))
	for _, config := range configuredExternalSources {
		source, err := queryExternalSource(ctx, tx, config.ProviderKey)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, nil
}

func queryExternalSource(ctx context.Context, tx pgx.Tx, provider string) (externalSourceRecord, error) {
	var source externalSourceRecord
	err := tx.QueryRow(ctx, `
		select provider_key, provider_label, sync_enabled, coalesce(last_test_status, ''), coalesce(last_test_summary, ''), last_test_at, actor_id, reason, updated_at
		from external_data_sources
		where provider_key = $1
	`, provider).Scan(&source.ProviderKey, &source.ProviderLabel, &source.SyncEnabled, &source.LastTestStatus, &source.LastTestSummary, &source.LastTestAt, &source.ActorID, &source.Reason, &source.UpdatedAt)
	if err != nil {
		return externalSourceRecord{}, err
	}
	rows, err := tx.Query(ctx, `
		select field_key, key_id, fingerprint, label, updated_at
		from external_provider_credentials
		where provider_key = $1
		order by field_key
	`, provider)
	if err != nil {
		return externalSourceRecord{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var credential providerCredentialRecord
		credential.Stored = true
		if err := rows.Scan(&credential.FieldKey, &credential.KeyID, &credential.Fingerprint, &credential.Label, &credential.UpdatedAt); err != nil {
			return externalSourceRecord{}, err
		}
		source.Credentials = append(source.Credentials, credential)
	}
	return source, rows.Err()
}

func queryAuthSettingsAudit(ctx context.Context, tx pgx.Tx) ([]authSettingsAudit, error) {
	rows, err := tx.Query(ctx, `
		select id, actor_id, actor_type, target_entity, target_id, reason, diff, created_at
		from audit_log
		where target_entity in (
			'auth_role_mapping',
			'auth_site_scope_mapping',
			'external_provider_credentials',
			'external_data_source',
			'external_data_source_test'
		)
		order by created_at desc
		limit 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []authSettingsAudit
	for rows.Next() {
		var event authSettingsAudit
		if err := rows.Scan(&event.ID, &event.ActorID, &event.ActorType, &event.TargetEntity, &event.TargetID, &event.Reason, &event.Diff, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func insertAuthSettingsAudit(ctx context.Context, tx pgx.Tx, actorID string, entity string, targetID string, reason string, diff map[string]any) error {
	rawDiff, err := json.Marshal(diff)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		insert into audit_log (actor_id, actor_type, target_entity, target_id, reason, diff, created_at)
		values ($1, 'dev_persona', $2, $3, $4, $5::jsonb, now())
	`, actorID, entity, targetID, reason, string(rawDiff))
	return err
}

func decodeAuthMappingWriteRequest(w http.ResponseWriter, r *http.Request, roles bool) (authMappingWriteRequest, error) {
	var request authMappingWriteRequest
	if err := decodeAdminAuthSettingsJSON(w, r, &request); err != nil {
		return request, err
	}
	return request, validateMappingRequest(request, roles)
}

func decodeMappingDeleteRequest(w http.ResponseWriter, r *http.Request) (authMappingDeleteRequest, error) {
	var request authMappingDeleteRequest
	if err := decodeAdminAuthSettingsJSON(w, r, &request); err != nil {
		return request, err
	}
	if strings.TrimSpace(request.Reason) == "" {
		return request, errors.New("change reason is required")
	}
	return request, nil
}

func decodeAuthPreviewRequest(w http.ResponseWriter, r *http.Request) (authPreviewRequest, error) {
	var request authPreviewRequest
	if err := decodeAdminAuthSettingsJSON(w, r, &request); err != nil {
		return request, err
	}
	if strings.TrimSpace(request.Email) == "" {
		return request, errors.New("candidate email is required")
	}
	if request.Attributes == nil {
		request.Attributes = map[string][]string{}
	}
	return request, nil
}

func decodeProviderCredentialWriteRequest(w http.ResponseWriter, r *http.Request) (providerCredentialWriteRequest, error) {
	var request providerCredentialWriteRequest
	if err := decodeAdminAuthSettingsJSON(w, r, &request); err != nil {
		return request, err
	}
	if strings.TrimSpace(request.Reason) == "" {
		return request, errors.New("change reason is required")
	}
	if len(request.Fields) == 0 {
		return request, errors.New("at least one credential field is required")
	}
	if request.Labels == nil {
		request.Labels = map[string]string{}
	}
	return request, nil
}

func decodeExternalSourceToggleRequest(w http.ResponseWriter, r *http.Request) (externalSourceToggleRequest, error) {
	var request externalSourceToggleRequest
	if err := decodeAdminAuthSettingsJSON(w, r, &request); err != nil {
		return request, err
	}
	if strings.TrimSpace(request.Reason) == "" {
		return request, errors.New("change reason is required")
	}
	return request, nil
}

func decodeProviderTestRequest(w http.ResponseWriter, r *http.Request) (providerTestRequest, error) {
	var request providerTestRequest
	if err := decodeAdminAuthSettingsJSON(w, r, &request); err != nil {
		return request, err
	}
	if strings.TrimSpace(request.Reason) == "" {
		return request, errors.New("change reason is required")
	}
	return request, nil
}

func decodeAdminAuthSettingsJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, adminAuthSettingsMaxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func validateMappingRequest(request authMappingWriteRequest, roles bool) error {
	if strings.TrimSpace(request.Reason) == "" {
		return errors.New("change reason is required")
	}
	if !slices.Contains([]string{"group", "ou", "attribute"}, request.SourceType) {
		return errors.New("source_type must be group, ou, or attribute")
	}
	if strings.TrimSpace(request.SourceValue) == "" {
		return errors.New("source_value is required")
	}
	if request.SourceType == "attribute" && len(sanitizeList(request.AttributeValues)) == 0 {
		return errors.New("attribute mappings require at least one attribute value")
	}
	if roles && len(sanitizeList(request.RoleKeys)) == 0 {
		return errors.New("role mappings require at least one role")
	}
	if !roles && len(sanitizeList(request.SiteCodes)) == 0 {
		return errors.New("site-scope mappings require at least one site code")
	}
	return nil
}

func buildAuthPreview(snapshot authSettingsSnapshot, request authPreviewRequest) map[string]any {
	policy := auth.DefaultPolicy()
	for _, record := range snapshot.RoleMappings {
		switch record.SourceType {
		case "group":
			policy.GroupRoleMappings = append(policy.GroupRoleMappings, auth.GroupRoleMapping{Group: record.SourceValue, Roles: record.RoleKeys})
		case "ou":
			policy.OURoleMappings = append(policy.OURoleMappings, auth.OURoleMapping{OU: record.SourceValue, Roles: record.RoleKeys})
		case "attribute":
			policy.AttributeRoleMappings = append(policy.AttributeRoleMappings, auth.AttributeRoleMapping{Attribute: record.SourceValue, Values: record.AttributeValues, Roles: record.RoleKeys})
		}
	}
	for _, record := range snapshot.SiteScopeMappings {
		policy.SiteScopeMappings = append(policy.SiteScopeMappings, auth.SiteScopeMapping{SourceType: record.SourceType, Source: record.SourceValue, Values: record.AttributeValues, Sites: record.SiteCodes})
	}
	decision := auth.EvaluateGoogleIdentity(policy, auth.GoogleIdentity{
		Email:      request.Email,
		Groups:     request.Groups,
		OUs:        request.OUs,
		Attributes: request.Attributes,
	})
	failures := []string{}
	if !decision.Authorized {
		failures = append(failures, decision.Reason)
	}
	return map[string]any{
		"authorized":          decision.Authorized,
		"email":               decision.Email,
		"roles":               decision.Roles,
		"site_scopes":         decision.SiteScopes,
		"validation_failures": failures,
		"production_login":    "disabled",
	}
}

func validateProviderCredentials(provider string, values map[string]string) (string, string) {
	if len(values) == 0 {
		return "failed", "No saved credential fields are available for the provider."
	}
	var missing []string
	for _, config := range configuredExternalSources {
		if config.ProviderKey != provider {
			continue
		}
		for _, field := range config.RequiredFields {
			if strings.TrimSpace(values[field]) == "" {
				missing = append(missing, field)
			}
		}
	}
	if len(missing) > 0 {
		return "failed", "Missing required encrypted credential fields: " + strings.Join(missing, ", ")
	}
	return "passed", "Saved encrypted credential fields were decrypted and validated with a read-only configuration probe; sync remains disabled unless toggled separately."
}

func encryptCredential(plaintext string) (string, string, string, error) {
	keyID, keyMaterial, err := activeCredentialEncryptionKey()
	if err != nil {
		return "", "", "", err
	}
	block, err := aes.NewCipher(keyMaterial)
	if err != nil {
		return "", "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	combined := append(nonce, ciphertext...)
	fingerprint := sha256.Sum256([]byte(plaintext))
	return "v1:" + keyID + ":" + base64.StdEncoding.EncodeToString(combined), keyID, hex.EncodeToString(fingerprint[:])[:16], nil
}

func decryptCredential(encrypted string) (string, error) {
	parts := strings.SplitN(encrypted, ":", 3)
	if len(parts) != 3 || parts[0] != "v1" {
		return "", errors.New("unsupported encrypted credential format")
	}
	_, keyMaterial, err := activeCredentialEncryptionKey()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(keyMaterial)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("encrypted credential payload is malformed")
	}
	nonce := raw[:gcm.NonceSize()]
	ciphertext := raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func activeCredentialEncryptionKey() (string, []byte, error) {
	raw := strings.TrimSpace(os.Getenv("ENCRYPTION_KEY"))
	if raw == "" {
		if currentAppEnvironment() != "development" {
			return "", nil, errors.New("ENCRYPTION_KEY is required before saving encrypted credentials")
		}
		raw = "dev-local-auth-settings:" + strings.Repeat("0", 32)
	}
	keyID := "default"
	secret := raw
	if before, after, ok := strings.Cut(raw, ":"); ok {
		keyID = strings.TrimSpace(before)
		secret = strings.TrimSpace(after)
	}
	if secret == "" {
		return "", nil, errors.New("ENCRYPTION_KEY is missing key material")
	}
	sum := sha256.Sum256([]byte(secret))
	return keyID, sum[:], nil
}

func sanitizeList(values []string) []string {
	seen := make(map[string]bool)
	var sanitized []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		sanitized = append(sanitized, value)
	}
	sortStrings(sanitized)
	return sanitized
}

func changedMappingFields(request authMappingWriteRequest, roles bool) []string {
	fields := []string{"source_type", "source_value"}
	if request.SourceType == "attribute" {
		fields = append(fields, "attribute_values")
	}
	if roles {
		fields = append(fields, "role_keys")
	} else {
		fields = append(fields, "site_codes")
	}
	return fields
}

func deleteMappingByID(records *[]authMappingRecord, id int64) bool {
	for index, record := range *records {
		if record.ID == id {
			*records = append((*records)[:index], (*records)[index+1:]...)
			return true
		}
	}
	return false
}

func upsertCredentialMeta(records []providerCredentialRecord, next providerCredentialRecord) []providerCredentialRecord {
	for index, record := range records {
		if record.FieldKey == next.FieldKey {
			records[index] = next
			return records
		}
	}
	return append(records, next)
}

func sanitizedCredentialLabel(label string, field string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return field + " stored"
	}
	if len(label) > 80 {
		label = label[:80]
	}
	return label
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sortStrings(keys)
	return keys
}

func configuredProvider(provider string) bool {
	return slices.ContainsFunc(configuredExternalSources, func(config externalSourceConfig) bool {
		return config.ProviderKey == provider
	})
}

func sortStrings(values []string) {
	slices.SortFunc(values, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
}
