package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devflow/internal/config"
)

func TestCreateIssue_Succeeds(t *testing.T) {
	respJSON := `{"key":"PROJ-123","fields":{"summary":"New task"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(respJSON))
	}))
	defer srv.Close()

	cfg := &config.JiraConfig{URL: srv.URL, Username: "u", Token: "t"}
	c := NewClient(cfg)

	issue, err := c.CreateIssue(CreateIssueOptions{ProjectKey: "PROJ", Summary: "New task"})
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if issue == nil || issue.Key != "PROJ-123" {
		t.Fatalf("unexpected issue returned: %+v", issue)
	}
}

func TestCreateIssue_RetryRemovesCustomFields(t *testing.T) {
	// Simulate server that returns 400 with errors on first attempt, then 201
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if r.URL.Path != "/rest/api/3/issue" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if call == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			// Return an errors map indicating customfield_10014 is invalid
			b, _ := json.Marshal(map[string]map[string]string{"errors": {"customfield_10014": "bad field"}})
			w.Write(b)
			return
		}
		// On retry, return created
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"key":"PROJ-456","fields":{"summary":"New task"}}`))
	}))
	defer srv.Close()

	cfg := &config.JiraConfig{URL: srv.URL, Username: "u", Token: "t"}
	c := NewClient(cfg)

	opts := CreateIssueOptions{ProjectKey: "PROJ", Summary: "Task", Epic: "E-1", StoryPoints: 3.0, Sprint: "S1", Team: "42"}
	issue, err := c.CreateIssue(opts)
	if err != nil {
		t.Fatalf("CreateIssue (with retry) failed: %v", err)
	}
	if issue == nil || !strings.HasPrefix(issue.Key, "PROJ-") {
		t.Fatalf("unexpected issue returned after retry: %+v", issue)
	}
	if call != 2 {
		t.Fatalf("expected 2 calls to server, got %d", call)
	}
}

func TestAddCommentAndRemoteLink_Extra(t *testing.T) {
	// Server will accept comment and remotelink
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/comment") {
			w.WriteHeader(http.StatusCreated)
			return
		}
		if strings.Contains(r.URL.Path, "/remotelink") {
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := &config.JiraConfig{URL: srv.URL, Username: "u", Token: "t"}
	c := NewClient(cfg)

	if err := c.AddComment("PROJ-1", "hello"); err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}
	if err := c.AddRemoteLink("PROJ-1", "https://example.com", "title", "summary"); err != nil {
		t.Fatalf("AddRemoteLink failed: %v", err)
	}
}
