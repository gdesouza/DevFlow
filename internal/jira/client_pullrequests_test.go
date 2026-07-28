package jira

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"devflow/internal/config"
	"devflow/internal/httpx"
)

func TestGetIssuePullRequests(t *testing.T) {
	server := httpx.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/dev-status/1.0/issue/detail" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("issueId") != "123 456" || r.URL.Query().Get("dataType") != "pullrequest" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(devStatusResponse{Detail: []devStatusDetail{{
			PullRequests: []PullRequestRef{{ID: "pr-1", Name: "Implement feature"}},
		}}})
	}))
	defer server.Close()

	client := NewClient(&config.JiraConfig{URL: server.URL, Username: "user", Token: "token"})
	prs, err := client.GetIssuePullRequests("123 456")
	if err != nil {
		t.Fatalf("GetIssuePullRequests failed: %v", err)
	}
	if len(prs) != 1 || prs[0].ID != "pr-1" {
		t.Fatalf("unexpected pull requests: %+v", prs)
	}
}

func TestGetIssuePullRequestsAPIError(t *testing.T) {
	server := httpx.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream unavailable"))
	}))
	defer server.Close()

	client := NewClient(&config.JiraConfig{URL: server.URL})
	_, err := client.GetIssuePullRequests("123")
	if err == nil || !strings.Contains(err.Error(), "502") || !strings.Contains(err.Error(), "upstream unavailable") {
		t.Fatalf("expected API error details, got %v", err)
	}
}

func TestGetIssuePullRequestsInvalidJSON(t *testing.T) {
	server := httpx.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewClient(&config.JiraConfig{URL: server.URL})
	_, err := client.GetIssuePullRequests("123")
	if err == nil || !strings.Contains(err.Error(), "failed to decode response") {
		t.Fatalf("expected decode error, got %v", err)
	}
}
