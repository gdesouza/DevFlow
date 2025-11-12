package bitbucket

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"devflow/internal/config"
)

// TestEmptyCommits ensures GetPullRequestCommits handles empty values gracefully
func TestEmptyCommits(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		body := `{"values":[]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
	}}
	c.httpClient = &http.Client{Transport: ft}

	commits, err := c.GetPullRequestCommits("repo", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("expected 0 commits, got %d", len(commits))
	}
}

// TestGetCommitStatuses_Non200 ensures non-200 from statuses returns an error
func TestGetCommitStatuses_Non200(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("server error")), Header: http.Header{"Content-Type": {"text/plain"}}}
	}}
	c.httpClient = &http.Client{Transport: ft}

	_, err := c.GetCommitStatuses("repo", "abcdef")
	if err == nil {
		t.Fatalf("expected error for non-200 response")
	}
}

// TestCreatePullRequest_Non201 ensures CreatePullRequest handles unexpected response codes
func TestCreatePullRequest_Non201(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		// Verify POST body contains title and source/destination
		if req.Method != "POST" {
			return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader("wrong method")), Header: http.Header{}}
		}
		b, _ := io.ReadAll(req.Body)
		var payload map[string]interface{}
		_ = json.Unmarshal(b, &payload)
		if payload["title"] == nil || payload["source"] == nil || payload["destination"] == nil {
			return &http.Response{StatusCode: 422, Body: io.NopCloser(strings.NewReader("missing fields")), Header: http.Header{"Content-Type": {"text/plain"}}}
		}
		// Return 400 to simulate API validation error
		return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader("bad request")), Header: http.Header{"Content-Type": {"text/plain"}}}
	}}
	c.httpClient = &http.Client{Transport: ft}

	_, err := c.CreatePullRequest("repo", "t", "d", "s", "m", []string{"r1"})
	if err == nil {
		t.Fatalf("expected error when API returns non-201")
	}
}

// TestGetRepositoriesPaged_Err ensures GetRepositoriesPaged surfaces API errors
func TestGetRepositoriesPaged_Err(t *testing.T) {
	cfg := &config.BitbucketConfig{Workspace: "w"}
	c := NewClient(cfg)
	c.rateLimiter = nil

	ft := &fakeTransport{handler: func(req *http.Request, call int) *http.Response {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("server error")), Header: http.Header{"Content-Type": {"text/plain"}}}
	}}
	c.httpClient = &http.Client{Transport: ft}

	_, _, err := c.GetRepositoriesPaged(1, 10)
	if err == nil {
		t.Fatalf("expected error from GetRepositoriesPaged when API fails")
	}
}
