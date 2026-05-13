package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"
)

type recordingDevFeatureFlagStore struct {
	mu              sync.Mutex
	state           map[string]map[devFeatureFlagTargetKey]bool
	audits          []devFeatureFlagAuditDelta
	snapshotErr     error
	snapshots       int
	lastSnapshotErr error
}

func newRecordingDevFeatureFlagStore() *recordingDevFeatureFlagStore {
	return &recordingDevFeatureFlagStore{state: initialDevFeatureFlagState()}
}

func (store *recordingDevFeatureFlagStore) Snapshot(ctx context.Context) (map[string]map[devFeatureFlagTargetKey]bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.snapshots++
	if err := ctx.Err(); err != nil {
		store.lastSnapshotErr = err
		return nil, err
	}
	if store.snapshotErr != nil {
		store.lastSnapshotErr = store.snapshotErr
		return nil, store.snapshotErr
	}
	return cloneDevFeatureFlagState(store.state), nil
}

func (store *recordingDevFeatureFlagStore) UpdateTargets(_ context.Context, flagKey string, updates []devFeatureFlagTargetUpdate, actorID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, ok := store.state[flagKey]; !ok {
		store.state[flagKey] = make(map[devFeatureFlagTargetKey]bool)
	}
	changedAt := time.Now().UTC()
	for _, update := range updates {
		target := devFeatureFlagTargetKey{TargetType: update.TargetType, TargetID: update.TargetID}
		beforeEnabled := store.state[flagKey][target]
		if beforeEnabled == update.Enabled {
			continue
		}
		store.state[flagKey][target] = update.Enabled
		store.audits = append(store.audits, devFeatureFlagAuditDelta{
			FlagKey:       flagKey,
			TargetType:    update.TargetType,
			TargetID:      update.TargetID,
			BeforeEnabled: beforeEnabled,
			AfterEnabled:  update.Enabled,
			ActorID:       actorID,
			ChangedAt:     changedAt,
		})
	}
	return nil
}

func (store *recordingDevFeatureFlagStore) targetEnabled(flagKey string, target devFeatureFlagTargetKey) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.state[flagKey][target]
}

func (store *recordingDevFeatureFlagStore) auditDeltas() []devFeatureFlagAuditDelta {
	store.mu.Lock()
	defer store.mu.Unlock()
	return slices.Clone(store.audits)
}

func (store *recordingDevFeatureFlagStore) snapshotCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.snapshots
}

func (store *recordingDevFeatureFlagStore) lastSnapshotError() error {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.lastSnapshotErr
}

func TestDevFeatureFlagHandlerPersistsAndAuditsTargets(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "")
	ResetDevFeatureFlagStateForTest()
	store := newRecordingDevFeatureFlagStore()
	devFeatureFlagStoreMu.Lock()
	devFeatureFlagStore = store
	devFeatureFlagStoreError = nil
	devFeatureFlagStoreMu.Unlock()
	t.Cleanup(ResetDevFeatureFlagStateForTest)

	handler := NewAppHandler(HealthDependencies{})
	itCookie := loginDevPersonaForPersistenceTest(t, handler, "it_admin")
	body, err := json.Marshal(map[string]any{
		"targets": []map[string]any{
			{"target_type": "persona", "target_id": "human_resources", "enabled": false},
			{"target_type": "site", "target_id": "district-office", "enabled": false},
		},
	})
	if err != nil {
		t.Fatalf("marshal update body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dev/feature-flags/onboarding", bytes.NewReader(body))
	req.AddCookie(itCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("feature flag update returned %d, want 200: %s", rec.Code, rec.Body.String())
	}

	if store.targetEnabled("onboarding", devFeatureFlagTargetKey{TargetType: "persona", TargetID: "human_resources"}) {
		t.Fatal("persistent store kept human_resources onboarding enabled, want disabled")
	}
	if store.targetEnabled("onboarding", devFeatureFlagTargetKey{TargetType: "site", TargetID: "district-office"}) {
		t.Fatal("persistent store kept district-office onboarding enabled, want disabled")
	}

	audits := store.auditDeltas()
	if len(audits) != 2 {
		t.Fatalf("audit delta count = %d, want 2: %#v", len(audits), audits)
	}
	for _, audit := range audits {
		if audit.FlagKey != "onboarding" || audit.ActorID != "it_admin" {
			t.Fatalf("audit identity fields = %#v, want onboarding by it_admin", audit)
		}
		if !audit.BeforeEnabled || audit.AfterEnabled {
			t.Fatalf("audit delta = %#v, want true to false", audit)
		}
		if audit.ChangedAt.IsZero() {
			t.Fatalf("audit timestamp was not captured: %#v", audit)
		}
	}

	devFeatureFlagStateMu.Lock()
	devFeatureFlagState = initialDevFeatureFlagState()
	devFeatureFlagStateLoaded = false
	devFeatureFlagStateLoadAttempted = false
	devFeatureFlagStateMu.Unlock()

	restartedHandler := NewAppHandler(HealthDependencies{})
	hrCookie := loginDevPersonaForPersistenceTest(t, restartedHandler, "human_resources")
	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
	sessionReq.AddCookie(hrCookie)
	sessionRec := httptest.NewRecorder()
	restartedHandler.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("restarted handler session returned %d, want 200: %s", sessionRec.Code, sessionRec.Body.String())
	}
	var session struct {
		AllowedRoutes []string `json:"allowed_routes"`
	}
	if err := json.NewDecoder(sessionRec.Body).Decode(&session); err != nil {
		t.Fatalf("decode restarted session: %v", err)
	}
	if slices.Contains(session.AllowedRoutes, "/onboarding") {
		t.Fatalf("restarted handler lost persisted target state; allowed routes = %#v", session.AllowedRoutes)
	}
}

func TestDevFeatureFlagLazyRefreshFailureIsCachedAndFailsClosed(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "")
	ResetDevFeatureFlagStateForTest()
	store := newRecordingDevFeatureFlagStore()
	store.snapshotErr = errors.New("snapshot unavailable")
	devFeatureFlagStoreMu.Lock()
	devFeatureFlagStore = store
	devFeatureFlagStoreError = nil
	devFeatureFlagStoreMu.Unlock()
	t.Cleanup(ResetDevFeatureFlagStateForTest)

	config := devPersonaConfigs["site_admin"]
	firstPayload := buildDevSessionPayload(context.Background(), config)
	if slices.Contains(firstPayload.AllowedRoutes, "/onboarding") {
		t.Fatalf("failed lazy refresh allowed default-enabled route; allowed routes = %#v", firstPayload.AllowedRoutes)
	}
	if store.snapshotCount() != 1 {
		t.Fatalf("snapshot count after first payload = %d, want 1", store.snapshotCount())
	}

	secondPayload := buildDevSessionPayload(context.Background(), config)
	if slices.Contains(secondPayload.AllowedRoutes, "/onboarding") {
		t.Fatalf("cached failed refresh allowed default-enabled route; allowed routes = %#v", secondPayload.AllowedRoutes)
	}
	if store.snapshotCount() != 1 {
		t.Fatalf("snapshot count after second payload = %d, want cached failed attempt without retry", store.snapshotCount())
	}
}

func TestDevFeatureFlagLazyRefreshUsesRequestContext(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "")
	ResetDevFeatureFlagStateForTest()
	store := newRecordingDevFeatureFlagStore()
	devFeatureFlagStoreMu.Lock()
	devFeatureFlagStore = store
	devFeatureFlagStoreError = nil
	devFeatureFlagStoreMu.Unlock()
	t.Cleanup(ResetDevFeatureFlagStateForTest)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if routeAllowed(ctx, devPersonaConfigs["site_admin"], "/onboarding") {
		t.Fatal("canceled lazy refresh allowed onboarding route, want fail closed")
	}
	if !errors.Is(store.lastSnapshotError(), context.Canceled) {
		t.Fatalf("snapshot error = %v, want request context cancellation", store.lastSnapshotError())
	}
}

func TestDevFeatureFlagUpdateSucceedsWhenPostCommitRefreshFails(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "")
	ResetDevFeatureFlagStateForTest()
	store := newRecordingDevFeatureFlagStore()
	store.snapshotErr = errors.New("snapshot unavailable after commit")
	devFeatureFlagStoreMu.Lock()
	devFeatureFlagStore = store
	devFeatureFlagStoreError = nil
	devFeatureFlagStoreMu.Unlock()
	t.Cleanup(ResetDevFeatureFlagStateForTest)

	err := updateDevFeatureFlagTargets(context.Background(), "onboarding", []devFeatureFlagTargetUpdate{
		{TargetType: "persona", TargetID: "human_resources", Enabled: false},
	}, "it_admin")
	if err != nil {
		t.Fatalf("update returned refresh failure after commit: %v", err)
	}
	if store.targetEnabled("onboarding", devFeatureFlagTargetKey{TargetType: "persona", TargetID: "human_resources"}) {
		t.Fatal("persistent store did not receive committed feature flag target update")
	}
}

func TestDevFeatureFlagUpdateSkipsUnchangedAuditEntries(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "")
	ResetDevFeatureFlagStateForTest()
	store := newRecordingDevFeatureFlagStore()
	devFeatureFlagStoreMu.Lock()
	devFeatureFlagStore = store
	devFeatureFlagStoreError = nil
	devFeatureFlagStoreMu.Unlock()
	t.Cleanup(ResetDevFeatureFlagStateForTest)

	err := updateDevFeatureFlagTargets(context.Background(), "onboarding", []devFeatureFlagTargetUpdate{
		{TargetType: "persona", TargetID: "human_resources", Enabled: true},
		{TargetType: "site", TargetID: "district-office", Enabled: false},
	}, "it_admin")
	if err != nil {
		t.Fatalf("update returned error: %v", err)
	}
	audits := store.auditDeltas()
	if len(audits) != 1 {
		t.Fatalf("audit delta count = %d, want only the changed target: %#v", len(audits), audits)
	}
	if audits[0].TargetType != "site" || audits[0].TargetID != "district-office" {
		t.Fatalf("audit delta = %#v, want district-office site change only", audits[0])
	}
}

func loginDevPersonaForPersistenceTest(t *testing.T, handler http.Handler, personaID string) *http.Cookie {
	t.Helper()
	body, err := json.Marshal(devLoginRequest{PersonaID: personaID})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login as %s returned %d, want 200: %s", personaID, rec.Code, rec.Body.String())
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == devSessionCookieName {
			return cookie
		}
	}
	t.Fatalf("login as %s did not set %s cookie", personaID, devSessionCookieName)
	return nil
}
