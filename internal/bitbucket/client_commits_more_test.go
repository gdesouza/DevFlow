package bitbucket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"devflow/internal/config"
)

func TestGetPullRequestCommits_Paginated(t *testing.T) {
	// Simulate API returning values + next (client should still return values)
	resp := CommitsResponse{Values: []Commit{{Hash: "aaa111", Message: "Commit A"}}}
	b, _ := json.Marshal(struct {
		CommitsResponse
		Next string `json:"next"`
	}{CommitsResponse: resp, Next: "https://api.bitbucket.org/2.0/repositories/workspace/repo/pullrequests/1/commits?page=2"})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/repositories/workspace/repo/pullrequests/1/commits" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}))
	defer server.Close()

	cfg := &config.BitbucketConfig{Workspace: "workspace", Token: "token"}
	c := NewClient(cfg)
	c.rateLimiter = nil
	c.baseURL = server.URL + "/2.0"

	commits, err := c.GetPullRequestCommits("repo", 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(commits) != 1 || commits[0].Hash != "aaa111" {
		t.Fatalf("unexpected commits returned: %+v", commits)
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
		w.Write(malformed)
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
