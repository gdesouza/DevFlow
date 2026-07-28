package jira

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"devflow/internal/config"
	"devflow/internal/httpx"
)

func TestSearchAll_DedupesAndFollowsTokens(t *testing.T) {
	calls := 0
	server := httpx.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		values := r.URL.Query()
		switch values.Get("pageToken") {
		case "":
			resp := SearchResponse{
				Issues:        []Issue{{Key: "ABC-1"}, {Key: "ABC-2"}},
				NextPageToken: "tok-1",
				IsLast:        false,
			}
			b, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(b)
		case "tok-1":
			resp := SearchResponse{
				Issues: []Issue{{Key: "ABC-2"}, {Key: "ABC-3"}},
				IsLast: true,
			}
			b, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(b)
		default:
			t.Fatalf("unexpected page token: %q", values.Get("pageToken"))
		}
	}))
	defer server.Close()

	client := NewClient(&config.JiraConfig{URL: server.URL, Username: "me", Token: "tok"})
	issues, err := client.SearchAll("project = ABC", true, 2, 0)
	if err != nil {
		t.Fatalf("SearchAll failed: %v", err)
	}
	if len(issues) != 3 {
		t.Fatalf("expected 3 unique issues, got %d", len(issues))
	}
	if issues[0].Key != "ABC-1" || issues[1].Key != "ABC-2" || issues[2].Key != "ABC-3" {
		t.Fatalf("unexpected issue order: %+v", []string{issues[0].Key, issues[1].Key, issues[2].Key})
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestSearchAll_StartAtPaging(t *testing.T) {
	calls := 0
	server := httpx.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		values := r.URL.Query()
		startAt := values.Get("startAt")
		resp := SearchResponse{
			StartAt:    0,
			MaxResults: 2,
			Total:      3,
		}
		switch startAt {
		case "":
			resp.Issues = []Issue{{Key: "ABC-1"}, {Key: "ABC-2"}}
		case "1", "2":
			resp.Issues = []Issue{{Key: "ABC-3"}}
		default:
			t.Fatalf("unexpected startAt: %q", startAt)
		}
		if startAt != "" {
			resp.StartAt = 2
		}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}))
	defer server.Close()

	client := NewClient(&config.JiraConfig{URL: server.URL, Username: "me", Token: "tok"})
	issues, err := client.SearchAll("project = ABC", true, 2, 0)
	if err != nil {
		t.Fatalf("SearchAll failed: %v", err)
	}
	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestUpdateIssue_ConvertsDescriptionAndSendsPUT(t *testing.T) {
	var received map[string]any
	server := httpx.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/rest/api/3/issue/ABC-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(&config.JiraConfig{URL: server.URL, Username: "me", Token: "tok"})
	err := client.UpdateIssue("ABC-1", map[string]any{
		"summary":     "Updated",
		"description": "first line\n\nsecond line",
	})
	if err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}

	fields, ok := received["fields"].(map[string]any)
	if !ok {
		t.Fatalf("missing fields in request: %#v", received)
	}
	desc, ok := fields["description"].(map[string]any)
	if !ok {
		t.Fatalf("description not converted to ADF: %#v", fields["description"])
	}
	content, ok := desc["content"].([]any)
	if !ok || len(content) != 3 {
		t.Fatalf("unexpected ADF content: %#v", desc["content"])
	}
}

func TestUpdateIssue_EmptyFieldsNoop(t *testing.T) {
	client := NewClient(&config.JiraConfig{URL: "http://unused", Username: "me", Token: "tok"})
	if err := client.UpdateIssue("ABC-1", map[string]any{}); err != nil {
		t.Fatalf("expected no-op update to succeed, got %v", err)
	}
}

func TestSearchAll_TerminatesOnRepeatedToken(t *testing.T) {
	calls := 0
	server := httpx.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		resp := SearchResponse{
			Issues:        []Issue{{Key: "ABC-1"}},
			NextPageToken: "loop",
		}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}))
	defer server.Close()

	client := NewClient(&config.JiraConfig{URL: server.URL, Username: "me", Token: "tok"})
	issues, err := client.SearchAll("project = ABC", true, 1, 0)
	if err == nil || !strings.Contains(err.Error(), "repeated nextPageToken") {
		t.Fatalf("expected repeated token error, got issues=%v err=%v", issues, err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected collected issues to be returned on error, got %d", len(issues))
	}
	if calls < 3 {
		t.Fatalf("expected repeated token retries, got %d", calls)
	}
}
