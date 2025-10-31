package bitbucket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"devflow/internal/config"
)

// mockServer helps simulate Bitbucket API responses for specific endpoints.
func mockServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h, ok := handlers[r.URL.Path]; ok {
			h(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestGetPullRequestCommits(t *testing.T) {
	// Prepare mock response
	commitsResp := CommitsResponse{Values: []Commit{
		{Hash: "abcdef123456", Message: "Add feature X", Date: "2025-10-20T12:00:00+00:00", Author: struct {
			Raw string `json:"raw"`
		}{Raw: "Alice <alice@example.com>"}},
		{Hash: "deadbeef98765", Message: "Refactor module", Date: "2025-10-20T13:00:00+00:00", Author: struct {
			Raw string `json:"raw"`
		}{Raw: "Bob <bob@example.com>"}},
	}}
	data, _ := json.Marshal(commitsResp)

	server := mockServer(t, map[string]http.HandlerFunc{
		"/2.0/repositories/workspace/repo/pullrequests/42/commits": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(data); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}

		},
	})
	t.Cleanup(server.Close)

	cfg := &config.BitbucketConfig{Workspace: "workspace", Token: "token"}
	client := NewClient(cfg)
	client.baseURL = server.URL + "/2.0"

	statuses, err := client.GetCommitStatuses("repo", "abcdef123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}
	if statuses[0].State != "SUCCESSFUL" {
		t.Errorf("expected first status SUCCESSFUL, got %s", statuses[0].State)
	}
}
