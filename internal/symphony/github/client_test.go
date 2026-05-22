package github_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	symphonygithub "github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/github"
)

type fakeRunner struct {
	responses map[string]string
	calls     []string
}

func (runner *fakeRunner) JSON(_ context.Context, args ...string) ([]byte, error) {
	key := strings.Join(args, "\x00")
	runner.calls = append(runner.calls, key)
	if response, ok := runner.responses[key]; ok {
		return []byte(response), nil
	}
	return []byte("[]"), nil
}

func TestListOpenIssuesPaginatesAndHydratesComments(t *testing.T) {
	firstPage := make([]map[string]any, 100)
	for index := range firstPage {
		firstPage[index] = map[string]any{"number": index + 1, "title": "Issue", "html_url": "https://example.invalid/issue"}
	}
	firstPage[0]["labels"] = []map[string]any{{"name": "agent-ready"}}
	secondPage := []map[string]any{{"number": 101, "title": "Old ready issue", "html_url": "https://example.invalid/101"}}
	page1, _ := json.Marshal(firstPage)
	page2, _ := json.Marshal(secondPage)
	comments, _ := json.Marshal([]map[string]any{{
		"id":         1,
		"body":       "Updated acceptance criteria",
		"html_url":   "https://example.invalid/comment",
		"created_at": "2026-05-22T00:00:00Z",
		"user":       map[string]any{"login": "owner"},
	}})
	runner := &fakeRunner{responses: map[string]string{
		"api\x00--method\x00GET\x00repos/o/r/issues\x00-f\x00state=open\x00-f\x00per_page=100\x00-f\x00page=1\x00-f\x00labels=agent-ready": string(page1),
		"api\x00--method\x00GET\x00repos/o/r/issues\x00-f\x00state=open\x00-f\x00per_page=100\x00-f\x00page=2\x00-f\x00labels=agent-ready": string(page2),
		"api\x00--method\x00GET\x00repos/o/r/issues/1/comments\x00-f\x00per_page=100\x00-f\x00page=1":                                      string(comments),
	}}

	issues, err := symphonygithub.ListOpenIssues(context.Background(), runner, "o", "r", "agent-ready")
	if err != nil {
		t.Fatalf("ListOpenIssues returned error: %v", err)
	}
	if len(issues) != 101 {
		t.Fatalf("expected paginated issue set, got %d", len(issues))
	}
	if len(issues[0].Comments) != 1 || issues[0].Comments[0].Body != "Updated acceptance criteria" {
		t.Fatalf("expected hydrated first issue comments, got %#v", issues[0].Comments)
	}
	if !called(runner.calls, "page=2") {
		t.Fatalf("expected second issue page request, calls: %#v", runner.calls)
	}
}

func called(calls []string, needle string) bool {
	for _, call := range calls {
		if strings.Contains(call, needle) {
			return true
		}
	}
	return false
}
