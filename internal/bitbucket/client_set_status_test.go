package bitbucket

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"devflow/internal/config"
)

func TestSetCommitStatus(t *testing.T) {
	var received struct {
		State       string `json:"state"`
		Key         string `json:"key"`
		Name        string `json:"name"`
		URL         string `json:"url"`
		Description string `json:"description"`
	}

	statusResp := CommitStatus{State: "SUCCESSFUL", Key: "ci/pipeline", Name: "CI Pipeline", URL: "https://ci.example.com/build/42", Description: "All tests passed", UpdatedOn: "2025-10-21T12:00:00+00:00", Type: "build"}
	respData, _ := json.Marshal(statusResp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/2.0/repositories/workspace/repo/commit/abcdef123456/statuses/build" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(bodyBytes, &received); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write(respData); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}

	}))
	defer server.Close()

	cfg := &config.BitbucketConfig{Workspace: "workspace", Token: "token"}
	client := NewClient(cfg)
	client.baseURL = server.URL + "/2.0"

	st, err := client.SetCommitStatus("repo", "abcdef123456", "SUCCESSFUL", "ci/pipeline", "CI Pipeline", "https://ci.example.com/build/42", "All tests passed")
	if err != nil {
		// include received for debugging
		t.Fatalf("expected no error, got %v (received: %+v)", err, received)
	}
	if st.State != "SUCCESSFUL" || st.Key != "ci/pipeline" {
		t.Errorf("unexpected response status: %+v", st)
	}
	// Validate request body mapping
	if received.State != "SUCCESSFUL" || received.Key != "ci/pipeline" || received.Name != "CI Pipeline" || received.URL != "https://ci.example.com/build/42" || received.Description != "All tests passed" {
		t.Errorf("request payload mismatch: %+v", received)
	}
}
