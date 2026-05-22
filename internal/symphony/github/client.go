package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// Issue is the minimal GitHub issue shape Symphony needs before it renders a
// worker prompt. Comments are intentionally full bodies, not summary counts,
// because issue comments can change acceptance criteria or branch guidance.
type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body,omitempty"`
	URL       string    `json:"url"`
	Labels    []string  `json:"labels,omitempty"`
	Comments  []Comment `json:"comments,omitempty"`
	UpdatedAt string    `json:"updated_at,omitempty"`
}

// Comment is one hydrated issue comment from the durable GitHub control plane.
type Comment struct {
	ID        int64  `json:"id"`
	Author    string `json:"author,omitempty"`
	Body      string `json:"body,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	URL       string `json:"url,omitempty"`
}

// Runner abstracts gh invocations so pagination and hydration can be tested
// without network access.
type Runner interface {
	JSON(ctx context.Context, args ...string) ([]byte, error)
}

// GHRunner uses the authenticated gh CLI as a migration bridge until the Go
// core owns direct REST/GraphQL calls.
type GHRunner struct {
	Dir string
}

// JSON executes gh and returns stdout so callers can decode typed responses.
func (runner GHRunner) JSON(ctx context.Context, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "gh", args...)
	command.Dir = runner.Dir
	output, err := command.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh %v: %s", args, string(exitErr.Stderr))
		}
		return nil, err
	}
	return output, nil
}

// ListOpenIssues returns all open issues for the repository, optionally scoped
// to a label, and hydrates every issue's comments before returning.
func ListOpenIssues(ctx context.Context, runner Runner, owner, repo, label string) ([]Issue, error) {
	rawIssues, err := paginateIssues(ctx, runner, owner, repo, label)
	if err != nil {
		return nil, err
	}
	issues := make([]Issue, 0, len(rawIssues))
	for _, raw := range rawIssues {
		if raw.PullRequest != nil {
			continue
		}
		comments, err := ListIssueComments(ctx, runner, owner, repo, raw.Number)
		if err != nil {
			return nil, fmt.Errorf("issue #%d comments: %w", raw.Number, err)
		}
		issues = append(issues, Issue{
			Number:    raw.Number,
			Title:     raw.Title,
			Body:      raw.Body,
			URL:       firstNonEmpty(raw.HTMLURL, raw.URL),
			Labels:    labelNames(raw.Labels),
			Comments:  comments,
			UpdatedAt: raw.UpdatedAt,
		})
	}
	return issues, nil
}

// ListIssueComments paginates issue comments so Symphony never dispatches with
// a summary count in place of the instructions humans added later.
func ListIssueComments(ctx context.Context, runner Runner, owner, repo string, number int) ([]Comment, error) {
	var comments []Comment
	for page := 1; ; page++ {
		var raw []apiComment
		if err := getJSON(ctx, runner, &raw, "api", "--method", "GET", fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, number), "-f", "per_page=100", "-f", "page="+strconv.Itoa(page)); err != nil {
			return nil, err
		}
		for _, item := range raw {
			comments = append(comments, Comment{
				ID:        item.ID,
				Author:    item.User.Login,
				Body:      item.Body,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
				URL:       item.HTMLURL,
			})
		}
		if len(raw) < 100 {
			break
		}
	}
	return comments, nil
}

func paginateIssues(ctx context.Context, runner Runner, owner, repo, label string) ([]apiIssue, error) {
	var issues []apiIssue
	for page := 1; ; page++ {
		args := []string{"api", "--method", "GET", fmt.Sprintf("repos/%s/%s/issues", owner, repo), "-f", "state=open", "-f", "per_page=100", "-f", "page=" + strconv.Itoa(page)}
		if label != "" {
			args = append(args, "-f", "labels="+label)
		}
		var raw []apiIssue
		if err := getJSON(ctx, runner, &raw, args...); err != nil {
			return nil, err
		}
		issues = append(issues, raw...)
		if len(raw) < 100 {
			break
		}
	}
	return issues, nil
}

func getJSON(ctx context.Context, runner Runner, target any, args ...string) error {
	output, err := runner.JSON(ctx, args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(output, target); err != nil {
		return fmt.Errorf("decode gh JSON for %v: %w", args, err)
	}
	return nil
}

func labelNames(labels []apiLabel) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		names = append(names, label.Name)
	}
	return names
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type apiIssue struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	URL         string          `json:"url"`
	HTMLURL     string          `json:"html_url"`
	UpdatedAt   string          `json:"updated_at"`
	Labels      []apiLabel      `json:"labels"`
	PullRequest json.RawMessage `json:"pull_request"`
}

type apiLabel struct {
	Name string `json:"name"`
}

type apiComment struct {
	ID        int64   `json:"id"`
	Body      string  `json:"body"`
	HTMLURL   string  `json:"html_url"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	User      apiUser `json:"user"`
}

type apiUser struct {
	Login string `json:"login"`
}
