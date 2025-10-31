package bitbucket

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devflow/internal/config"
)

func TestCreatePullRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/2.0/repositories/workspace/repo/pullrequests" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Read body and validate fields
		b, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		_ = json.Unmarshal(b, &payload)
		if payload["title"] == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Return created PR
		resp := map[string]interface{}{"id": 123, "title": payload["title"], "state": "OPEN"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.BitbucketConfig{Workspace: "workspace"}
	c := NewClient(cfg)
	c.rateLimiter = nil
	c.baseURL = server.URL + "/2.0"

	pr, err := c.CreatePullRequest("repo", "My PR", "desc", "feature", "main", []string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.ID != 123 || pr.Title != "My PR" {
		t.Fatalf("unexpected PR returned: %+v", pr)
	}
}

func TestGetCommitStatuses_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/commit/abcdef/statuses") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"values":[]}`)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.BitbucketConfig{Workspace: "workspace"}
	c := NewClient(cfg)
	c.rateLimiter = nil
	c.baseURL = server.URL + "/2.0"

	ss, err := c.GetCommitStatuses("repo", "abcdef")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ss) != 0 {
		t.Fatalf("expected 0 statuses, got %d", len(ss))
	}
}

func TestGetPullRequestCommits_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	cfg := &config.BitbucketConfig{Workspace: "workspace"}
	c := NewClient(cfg)
	c.rateLimiter = nil
	c.baseURL = server.URL + "/2.0"

	_, err := c.GetPullRequestCommits("repo", 999)
	if err == nil {
		t.Fatalf("expected error for non-200 response")
	}
}

func TestGetRepositoriesPaged_SuccessPages(t *testing.T) {
	// Use a server that handles both the count request and page requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.RawQuery
		path := r.URL.Path
		// getTotalRepositoryCount requests: /2.0/repositories/workspace?page=1&pagelen=1
		if strings.HasPrefix(path, "/2.0/repositories/workspace") && strings.Contains(q, "pagelen=1") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"values":[{"name":"r1","full_name":"workspace/r1"}], "next":""}`))
			return
		}
		// first page (getFirstPageWithTotal): page=1&pagelen=size
		if strings.HasPrefix(path, "/2.0/repositories/workspace") && strings.Contains(q, "page=1") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"values":[{"name":"a","full_name":"workspace/a"}], "next":""}`))
			return
		}
		// subsequent page e.g., page=2
		if strings.HasPrefix(path, "/2.0/repositories/workspace") && strings.Contains(q, "page=2") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"values":[{"name":"b","full_name":"workspace/b"}], "next":""}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.BitbucketConfig{Workspace: "workspace"}
	c := NewClient(cfg)
	c.rateLimiter = nil
	c.baseURL = server.URL + "/2.0"

	// page 0 -> getFirstPageWithTotal path
	vals, total, err := c.GetRepositoriesPaged(0, 10)
	if err != nil {
		t.Fatalf("unexpected error for page 0: %v", err)
	}
	if len(vals) != 1 || total < 1 {
		t.Fatalf("unexpected page0 result: vals=%v total=%d", vals, total)
	}

	// page 1 (apiPage >0) -> should call getTotalRepositoryCount then page=2
	vals2, total2, err := c.GetRepositoriesPaged(1, 10)
	if err != nil {
		t.Fatalf("unexpected error for page 1: %v", err)
	}
	if len(vals2) != 1 || total2 < 1 {
		t.Fatalf("unexpected page1 result: vals=%v total=%d", vals2, total2)
	}
}
