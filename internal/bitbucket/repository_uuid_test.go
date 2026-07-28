package bitbucket

import (
	"net/http"
	"testing"

	"devflow/internal/config"
)

func TestGetRepositoryByUUID(t *testing.T) {
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repositories/workspace" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"size":1,"values":[{"uuid":"{61c4d0b9-7a12-469c-a05a-61284c5e363e}","name":"mapping-service","full_name":"workspace/mapping-service"}]}`))
	}))
	defer server.Close()

	client := NewClient(&config.BitbucketConfig{Workspace: "workspace"})
	client.rateLimiter = nil
	client.baseURL = server.URL

	repository, err := client.GetRepository("{61c4d0b9-7a12-469c-a05a-61284c5e363e}")
	if err != nil {
		t.Fatalf("GetRepository by UUID failed: %v", err)
	}
	if repository.Name != "mapping-service" {
		t.Fatalf("repository name = %q, want %q", repository.Name, "mapping-service")
	}
}
