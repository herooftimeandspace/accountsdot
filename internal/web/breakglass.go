package web

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	breakglassSessionCookiePrefix = "breakglass:"
	breakglassDefaultPersonaID    = "it_admin"
	breakglassMaxBodyBytes        = 4 * 1024
)

var breakglassDefaultAllowedCIDRs = []string{
	"10.23.0.0/16",
	"10.19.100.0/24",
}

type breakglassLoginRequest struct {
	AccountID string `json:"account_id"`
	Token     string `json:"token"`
}

type breakglassAccount struct {
	AccountID string
	PersonaID string
	TokenHash string
}

type breakglassAuditEvent struct {
	AccountID     string    `json:"account_id"`
	Action        string    `json:"action"`
	Outcome       string    `json:"outcome"`
	SourceIP      string    `json:"source_ip"`
	PersonaID     string    `json:"persona_id,omitempty"`
	FailureCode   string    `json:"failure_code,omitempty"`
	RecordedAt    time.Time `json:"recorded_at"`
	RequestID     string    `json:"request_id,omitempty"`
	TargetSession string    `json:"target_session,omitempty"`
}

type breakglassAuditStorage interface {
	RecordBreakglassAudit(context.Context, breakglassAuditEvent) error
}

type memoryBreakglassAuditStore struct{}

type postgresBreakglassAuditStore struct {
	pool *pgxpool.Pool
}

var (
	breakglassAuditMu         sync.Mutex
	breakglassAuditEvents     []breakglassAuditEvent
	breakglassAuditStoreMu    sync.Mutex
	breakglassAuditStore      breakglassAuditStorage
	breakglassAuditStoreError error
)

// handleBreakglassLogin is the local emergency sign-in route used when Google
// SAML is unavailable in DEV or staging. The React DEV persona switcher still
// posts to /api/v1/dev/login; this route accepts only named breakglass accounts,
// verifies a per-account token hash from the environment, checks the request
// source address against configured CIDRs, writes an audit event, and then
// issues a cookie-scoped IT Admin session for the local emergency account.
func handleBreakglassLogin(w http.ResponseWriter, r *http.Request) {
	if !breakglassModeEnabled() || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	request, err := decodeBreakglassLoginRequest(w, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "invalid_request",
			"message": "Breakglass login requires account_id and token.",
		})
		return
	}
	accountID := normalizeBreakglassAccountID(request.AccountID)
	sourceIP := sourceIPForBreakglass(r)
	requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))

	account, ok := configuredBreakglassAccounts()[accountID]
	if !ok {
		recordBreakglassAudit(r.Context(), breakglassAuditEvent{
			AccountID:   accountID,
			Action:      "login_attempt",
			Outcome:     "denied",
			SourceIP:    sourceIP.String(),
			FailureCode: "unknown_account",
			RecordedAt:  time.Now().UTC(),
			RequestID:   requestID,
		})
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "breakglass_denied",
			"message": "Breakglass account is not configured.",
		})
		return
	}

	if sourceIP == nil || !breakglassSourceAllowed(sourceIP) {
		recordBreakglassAudit(r.Context(), breakglassAuditEvent{
			AccountID:   account.AccountID,
			Action:      "login_attempt",
			Outcome:     "denied",
			SourceIP:    sourceIP.String(),
			PersonaID:   account.PersonaID,
			FailureCode: "source_address_denied",
			RecordedAt:  time.Now().UTC(),
			RequestID:   requestID,
		})
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "breakglass_source_denied",
			"message": "Breakglass login is not allowed from this source address.",
		})
		return
	}

	if !breakglassTokenMatches(account.TokenHash, request.Token) {
		recordBreakglassAudit(r.Context(), breakglassAuditEvent{
			AccountID:   account.AccountID,
			Action:      "login_attempt",
			Outcome:     "denied",
			SourceIP:    sourceIP.String(),
			PersonaID:   account.PersonaID,
			FailureCode: "token_denied",
			RecordedAt:  time.Now().UTC(),
			RequestID:   requestID,
		})
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "breakglass_denied",
			"message": "Breakglass credentials were not accepted.",
		})
		return
	}

	config, ok := devPersonaConfigs[account.PersonaID]
	if !ok {
		recordBreakglassAudit(r.Context(), breakglassAuditEvent{
			AccountID:   account.AccountID,
			Action:      "login_attempt",
			Outcome:     "denied",
			SourceIP:    sourceIP.String(),
			PersonaID:   account.PersonaID,
			FailureCode: "persona_not_configured",
			RecordedAt:  time.Now().UTC(),
			RequestID:   requestID,
		})
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "breakglass_configuration_invalid",
			"message": "Breakglass account maps to an unavailable local persona.",
		})
		return
	}

	writeBreakglassSessionCookie(w, account.AccountID)
	now := time.Now().UTC()
	recordBreakglassAudit(r.Context(), breakglassAuditEvent{
		AccountID:     account.AccountID,
		Action:        "login_attempt",
		Outcome:       "allowed",
		SourceIP:      sourceIP.String(),
		PersonaID:     account.PersonaID,
		RecordedAt:    now,
		RequestID:     requestID,
		TargetSession: "cookie:" + devSessionCookieName,
	})
	recordBreakglassAudit(r.Context(), breakglassAuditEvent{
		AccountID:     account.AccountID,
		Action:        "access_granted",
		Outcome:       "allowed",
		SourceIP:      sourceIP.String(),
		PersonaID:     account.PersonaID,
		RecordedAt:    now,
		RequestID:     requestID,
		TargetSession: "cookie:" + devSessionCookieName,
	})
	writeJSON(w, http.StatusOK, buildBreakglassSessionPayload(r.Context(), config, account.AccountID))
}

// decodeBreakglassLoginRequest reads the local emergency login body for
// handleBreakglassLogin. It rejects oversized payloads, unknown fields, and
// trailing JSON so credentials cannot be hidden in ignored request fields.
func decodeBreakglassLoginRequest(w http.ResponseWriter, r *http.Request) (breakglassLoginRequest, error) {
	var request breakglassLoginRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, breakglassMaxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return breakglassLoginRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return breakglassLoginRequest{}, err
		}
		return breakglassLoginRequest{}, errors.New("request body must contain one JSON object")
	}
	if normalizeBreakglassAccountID(request.AccountID) == "" || request.Token == "" {
		return breakglassLoginRequest{}, errors.New("account_id and token are required")
	}
	return request, nil
}

// configuredBreakglassAccounts builds the named-account registry from
// BREAKGLASS_ACCOUNTS. Each account id must have a matching
// BREAKGLASS_TOKEN_SHA256_<SANITIZED_ACCOUNT_ID> environment variable; token
// material itself is never logged, documented, or stored in process globals.
func configuredBreakglassAccounts() map[string]breakglassAccount {
	accounts := map[string]breakglassAccount{}
	for _, rawAccountID := range strings.Split(os.Getenv("BREAKGLASS_ACCOUNTS"), ",") {
		accountID := normalizeBreakglassAccountID(rawAccountID)
		if accountID == "" {
			continue
		}
		tokenHash := strings.TrimSpace(os.Getenv(breakglassTokenHashEnvName(accountID)))
		if tokenHash == "" {
			continue
		}
		accounts[accountID] = breakglassAccount{
			AccountID: accountID,
			PersonaID: breakglassDefaultPersonaID,
			TokenHash: strings.ToLower(tokenHash),
		}
	}
	return accounts
}

var breakglassEnvNameCleaner = regexp.MustCompile(`[^A-Za-z0-9]+`)

func breakglassTokenHashEnvName(accountID string) string {
	sanitized := strings.Trim(breakglassEnvNameCleaner.ReplaceAllString(accountID, "_"), "_")
	return "BREAKGLASS_TOKEN_SHA256_" + strings.ToUpper(sanitized)
}

func normalizeBreakglassAccountID(accountID string) string {
	return strings.ToLower(strings.TrimSpace(accountID))
}

// sourceIPForBreakglass returns the concrete peer IP used for application-level
// source restrictions. The first X-Forwarded-For hop is honored so staging can
// sit behind an intranet reverse proxy, but the deployment must keep that proxy
// header trusted only at the private ingress boundary.
func sourceIPForBreakglass(r *http.Request) net.IP {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if ip := net.ParseIP(strings.TrimSpace(parts[0])); ip != nil {
			return ip
		}
	}
	host := r.RemoteAddr
	if splitHost, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = splitHost
	}
	return net.ParseIP(strings.TrimSpace(host))
}

func breakglassSourceAllowed(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, network := range breakglassAllowedNetworks() {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func breakglassAllowedNetworks() []*net.IPNet {
	rawCIDRs := breakglassDefaultAllowedCIDRs
	if override := strings.TrimSpace(os.Getenv("BREAKGLASS_ALLOWED_CIDRS")); override != "" {
		rawCIDRs = strings.Split(override, ",")
	}
	networks := make([]*net.IPNet, 0, len(rawCIDRs))
	for _, rawCIDR := range rawCIDRs {
		_, network, err := net.ParseCIDR(strings.TrimSpace(rawCIDR))
		if err == nil {
			networks = append(networks, network)
		}
	}
	return networks
}

func breakglassTokenMatches(expectedHash string, token string) bool {
	decodedExpected, err := hex.DecodeString(strings.TrimSpace(expectedHash))
	if err != nil || len(decodedExpected) != sha256.Size {
		return false
	}
	actual := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(decodedExpected, actual[:]) == 1
}

func breakglassModeEnabled() bool {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	return mode == "development" || mode == "staging"
}

func writeBreakglassSessionCookie(w http.ResponseWriter, accountID string) {
	writeDevSessionCookieValue(w, breakglassSessionCookiePrefix+accountID)
}

func buildBreakglassSessionPayload(ctx context.Context, config devPersonaConfig, accountID string) devSessionPayload {
	payload := buildDevSessionPayload(ctx, config)
	payload.Environment = strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	payload.AuthenticationMode = "breakglass"
	payload.BreakglassAccountID = accountID
	return payload
}

func recordBreakglassAudit(ctx context.Context, event breakglassAuditEvent) {
	if event.RecordedAt.IsZero() {
		event.RecordedAt = time.Now().UTC()
	}
	store, err := currentBreakglassAuditStore(ctx)
	if err != nil {
		return
	}
	_ = store.RecordBreakglassAudit(ctx, event)
}

func currentBreakglassAuditStore(ctx context.Context) (breakglassAuditStorage, error) {
	breakglassAuditStoreMu.Lock()
	defer breakglassAuditStoreMu.Unlock()
	if breakglassAuditStore != nil || breakglassAuditStoreError != nil {
		return breakglassAuditStore, breakglassAuditStoreError
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		breakglassAuditStore = memoryBreakglassAuditStore{}
		return breakglassAuditStore, nil
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		breakglassAuditStoreError = err
		return nil, err
	}
	breakglassAuditStore = postgresBreakglassAuditStore{pool: pool}
	return breakglassAuditStore, nil
}

// RecordBreakglassAudit appends sanitized local emergency auth events for
// database-free DEV runs. Tests inspect this store to verify login, denial, and
// sign-out evidence without introducing committed credentials or fixtures.
func (memoryBreakglassAuditStore) RecordBreakglassAudit(_ context.Context, event breakglassAuditEvent) error {
	breakglassAuditMu.Lock()
	defer breakglassAuditMu.Unlock()
	breakglassAuditEvents = append(breakglassAuditEvents, event)
	return nil
}

// RecordBreakglassAudit persists local emergency auth events to audit_log when
// DATABASE_URL is configured. The write uses db.WithRetry, stores only
// non-secret account/source/outcome metadata, and deliberately avoids recording
// submitted token material.
func (store postgresBreakglassAuditStore) RecordBreakglassAudit(ctx context.Context, event breakglassAuditEvent) error {
	return db.WithRetry(ctx, store.pool, func(tx pgx.Tx) error {
		diff, err := json.Marshal(event)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			insert into audit_log (actor_id, actor_type, request_id, target_entity, target_id, reason, diff, created_at)
			values ($1, 'breakglass_local_account', $2, 'session', $3, $4, $5::jsonb, $6)
		`, event.AccountID, nullIfBlank(event.RequestID), event.TargetSession, "breakglass_"+event.Action+"_"+event.Outcome, string(diff), event.RecordedAt)
		return err
	})
}

func nullIfBlank(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// ResetBreakglassAuditForTest clears the process-local audit store and cached
// storage backend so focused handler tests can assert exactly which sanitized
// events a breakglass login or logout generated.
func ResetBreakglassAuditForTest() {
	breakglassAuditMu.Lock()
	breakglassAuditEvents = nil
	breakglassAuditMu.Unlock()
	breakglassAuditStoreMu.Lock()
	breakglassAuditStore = nil
	breakglassAuditStoreError = nil
	breakglassAuditStoreMu.Unlock()
}

// BreakglassAuditEventsForTest returns process-local breakglass audit events in
// recorded order. It is intentionally test-only evidence for the memory-backed
// DEV path; staging should use audit_log when DATABASE_URL is configured.
func BreakglassAuditEventsForTest() []breakglassAuditEvent {
	breakglassAuditMu.Lock()
	defer breakglassAuditMu.Unlock()
	events := append([]breakglassAuditEvent(nil), breakglassAuditEvents...)
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].RecordedAt.Before(events[j].RecordedAt)
	})
	return events
}

func domainGateAllowsDashboardEmail(email string, localBreakglass bool) bool {
	if localBreakglass {
		return true
	}
	normalized := strings.ToLower(strings.TrimSpace(email))
	if strings.HasSuffix(normalized, "@stu.wusd.org") {
		return false
	}
	return strings.HasSuffix(normalized, "@wusd.org") ||
		strings.HasSuffix(normalized, "@it.wusd.org") ||
		strings.HasSuffix(normalized, "@staff.wusd.org")
}
