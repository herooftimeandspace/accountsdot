package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/web"
)

// TestOpenAPISpecSurfaceLabels exercises the public OpenAPI route registered
// by NewAppHandler. It decodes the served generated spec and verifies that
// clients can distinguish callable runtime discovery, planned DB-backed
// runtime APIs, accepted no-op write boundaries, and DEV-only mock endpoints
// without relying on separate route-name conventions.
func TestOpenAPISpecSurfaceLabels(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	rec := httptest.NewRecorder()

	web.NewAppHandler(web.HealthDependencies{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("openapi route returned %d, want 200", rec.Code)
	}
	var spec struct {
		OpenAPI string                                     `json:"openapi"`
		Paths   map[string]map[string]openAPITestOperation `json:"paths"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&spec); err != nil {
		t.Fatalf("decode openapi response: %v", err)
	}
	if spec.OpenAPI != "3.1.0" {
		t.Fatalf("openapi version = %q, want 3.1.0", spec.OpenAPI)
	}

	assertOperationLabel(t, spec.Paths, "/api/v1/openapi.json", "get", "db-backed-runtime", "public", "read-only")
	assertOperationLabel(t, spec.Paths, "/api/v1/session/me", "get", "db-backed-runtime-planned", "edge-auth-planned", "read-only")
	assertOperationLabel(t, spec.Paths, "/api/v1/workflows/{workflow_run_id}/retry", "post", "accepted-no-op", "staff-session-required-planned", "planned-db-write-boundary")
	assertOperationLabel(t, spec.Paths, "/api/v1/dev/session", "get", "dev-mock", "dev-cookie-or-anonymous", "read-only")
	assertOperationLabel(t, spec.Paths, "/api/v1/dev/feature-flags/{key}", "put", "dev-mock-or-local-db", "it-admin-required", "dev-memory-or-local-db-write")
	assertOperationResponse(t, spec.Paths, "/api/v1/dev/room-moves/drafts/{id}/cancel", "post", "200", "#/components/schemas/DevRoomMoveDraftResponse")
	assertOperationResponse(t, spec.Paths, "/api/v1/dev/room-moves/drafts/{id}/schedule", "post", "200", "#/components/schemas/DevRoomMoveDraftResponse")
	assertOperationErrorResponses(t, spec.Paths, "/api/v1/dev/room-moves/drafts/{id}/schedule", "post", []string{"400", "401", "403", "404", "409"})
	assertOperationHasNoRequestBody(t, spec.Paths, "/api/v1/dev/room-moves/drafts/{id}/schedule", "post")
	assertOperationNoContentResponse(t, spec.Paths, "/api/v1/dev/room-moves/drafts/{id}", "delete", "204")
	assertOperationSecurity(t, spec.Paths, "/api/v1/session/me", "get", false)
	assertOperationSecurity(t, spec.Paths, "/api/v1/breakglass/login", "post", false)
	assertOperationSecurity(t, spec.Paths, "/api/v1/dev/room-moves/drafts/{id}/schedule", "post", true)
}

// assertOperationLabel keeps TestOpenAPISpecSurfaceLabels focused on the API
// contract behavior rather than nested map checks. A failure identifies the
// exact OpenAPI path and method whose surface/auth/write metadata drifted.
func assertOperationLabel(t *testing.T, paths map[string]map[string]openAPITestOperation, path string, method string, wantSurface string, wantAuth string, wantWriteBoundary string) {
	t.Helper()

	operation, ok := paths[path][method]
	if !ok {
		t.Fatalf("missing OpenAPI operation %s %s", method, path)
	}
	if operation.Surface != wantSurface || operation.Auth != wantAuth || operation.WriteBoundary != wantWriteBoundary {
		t.Fatalf("%s %s labels = surface:%q auth:%q write:%q, want surface:%q auth:%q write:%q", method, path, operation.Surface, operation.Auth, operation.WriteBoundary, wantSurface, wantAuth, wantWriteBoundary)
	}
}

type openAPITestOperation struct {
	Surface       string `json:"x-wizard-surface"`
	Auth          string `json:"x-wizard-auth"`
	WriteBoundary string `json:"x-wizard-write-boundary"`
	RequestBody   any    `json:"requestBody"`
	Responses     map[string]struct {
		Content map[string]struct {
			Schema struct {
				Ref string `json:"$ref"`
			} `json:"schema"`
		} `json:"content"`
	} `json:"responses"`
	Security []map[string][]string `json:"security"`
}

// openAPIOperation centralizes lookup for response/status assertions so the
// regression checks below fail with the exact method and path that drifted.
func openAPIOperation(t *testing.T, paths map[string]map[string]openAPITestOperation, path string, method string) openAPITestOperation {
	t.Helper()

	operation, ok := paths[path][method]
	if !ok {
		t.Fatalf("missing OpenAPI operation %s %s", method, path)
	}
	return operation
}

// assertOperationResponse verifies endpoint-specific success status codes and
// response schemas. This protects DEV mock operations that return 200/201/204
// from being collapsed back to a generated 202 write response.
func assertOperationResponse(t *testing.T, paths map[string]map[string]openAPITestOperation, path string, method string, status string, schemaRef string) {
	t.Helper()

	operation := openAPIOperation(t, paths, path, method)
	response, ok := operation.Responses[status]
	if !ok {
		t.Fatalf("%s %s missing success response %s; responses: %#v", method, path, status, operation.Responses)
	}
	gotRef := response.Content["application/json"].Schema.Ref
	if gotRef != schemaRef {
		t.Fatalf("%s %s response %s schema = %q, want %q", method, path, status, gotRef, schemaRef)
	}
}

// assertOperationNoContentResponse checks success responses such as DELETE
// /room-moves/drafts/{id}, where the handler writes a status with no JSON body.
func assertOperationNoContentResponse(t *testing.T, paths map[string]map[string]openAPITestOperation, path string, method string, status string) {
	t.Helper()

	operation := openAPIOperation(t, paths, path, method)
	response, ok := operation.Responses[status]
	if !ok {
		t.Fatalf("%s %s missing no-content response %s; responses: %#v", method, path, status, operation.Responses)
	}
	if len(response.Content) != 0 {
		t.Fatalf("%s %s response %s content = %#v, want no body", method, path, status, response.Content)
	}
}

// assertOperationErrorResponses verifies that generated write operations list
// the client-error statuses actually emitted by handlers instead of a generic
// validation-only placeholder.
func assertOperationErrorResponses(t *testing.T, paths map[string]map[string]openAPITestOperation, path string, method string, statuses []string) {
	t.Helper()

	operation := openAPIOperation(t, paths, path, method)
	for _, status := range statuses {
		if _, ok := operation.Responses[status]; !ok {
			t.Fatalf("%s %s missing error response %s; responses: %#v", method, path, status, operation.Responses)
		}
	}
}

// assertOperationHasNoRequestBody protects route contracts for transition
// endpoints whose handlers intentionally ignore request JSON.
func assertOperationHasNoRequestBody(t *testing.T, paths map[string]map[string]openAPITestOperation, path string, method string) {
	t.Helper()

	operation := openAPIOperation(t, paths, path, method)
	if operation.RequestBody != nil {
		t.Fatalf("%s %s requestBody = %#v, want omitted", method, path, operation.RequestBody)
	}
}

// assertOperationSecurity distinguishes cookie-backed routes from public,
// anonymous, development-only, edge-auth-planned, and token-bootstrap APIs.
func assertOperationSecurity(t *testing.T, paths map[string]map[string]openAPITestOperation, path string, method string, wantCookie bool) {
	t.Helper()

	operation := openAPIOperation(t, paths, path, method)
	hasCookie := len(operation.Security) == 1 && operation.Security[0] != nil
	if hasCookie {
		_, hasCookie = operation.Security[0]["sessionCookie"]
	}
	if hasCookie != wantCookie {
		t.Fatalf("%s %s cookie security = %t, want %t; security: %#v", method, path, hasCookie, wantCookie, operation.Security)
	}
}
