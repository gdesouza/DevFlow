package bitbucket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"devflow/internal/config"
)

func TestGetPullRequestCommits_Paginated(t *testing.T) {
	// Simulate API returning two pages of commits
	var serverURL string

	page1 := CommitsResponse{
		Values: []Commit{{Hash: "aaa111", Message: "Commit A"}},
	}
	page2 := CommitsResponse{
		Values: []Commit{{Hash: "bbb222", Message: "Commit B"}},
		Next:   "", // Last page
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if r.URL.Path == "/2.0/repositories/workspace/repo/pullrequests/1/commits" && r.URL.RawQuery == "" {
			// First page - set Next to point to our test server
			page1.Next = serverURL + "/2.0/repositories/workspace/repo/pullrequests/1/commits?page=2"
			if err := json.NewEncoder(w).Encode(page1); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		} else if r.URL.Path == "/2.0/repositories/workspace/repo/pullrequests/1/commits" && r.URL.RawQuery == "page=2" {
			// Second page
			if err := json.NewEncoder(w).Encode(page2); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	cfg := &config.BitbucketConfig{Workspace: "workspace", Token: "token"}
	c := NewClient(cfg)
	c.rateLimiter = nil
	c.baseURL = server.URL + "/2.0"

	commits, err := c.GetPullRequestCommits("repo", 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	if commits[0].Hash != "aaa111" {
		t.Errorf("expected first commit hash 'aaa111', got '%s'", commits[0].Hash)
	}
	if commits[1].Hash != "bbb222" {
		t.Errorf("expected second commit hash 'bbb222', got '%s'", commits[1].Hash)
	}
}

func TestGetPullRequestCommits_MalformedCommit(t *testing.T) {
	// Return JSON where a commit hash is a number (type mismatch -> decode error)
	malformed := []byte(`{"values":[{"hash":12345,"message":"Bad"}]}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/repositories/workspace/repo/pullrequests/100/commits" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(malformed); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.BitbucketConfig{Workspace: "workspace", Token: "token"}
	c := NewClient(cfg)
	c.rateLimiter = nil
	c.baseURL = server.URL + "/2.0"

	_, err := c.GetPullRequestCommits("repo", 100)
	if err == nil {
		t.Fatalf("expected decode error for malformed commit JSON, got nil")
	}
}
