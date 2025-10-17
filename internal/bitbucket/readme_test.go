package bitbucket

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"devflow/internal/config"
)

func TestGetRepositoryReadme(t *testing.T) {
	cases := []struct {
		name           string
		files          map[string]string // path -> content
		expectFileName string
		expectContent  string
	}{
		{
			name: "MarkdownReadmeFound",
			files: map[string]string{
				"/repositories/ws/repo/src/HEAD/README.md": "# Title\nBody",
			},
			expectFileName: "README.md",
			expectContent:  "# Title\nBody",
		},
		{
			name: "PlainReadmeFound",
			files: map[string]string{
				"/repositories/ws/repo/src/HEAD/README": "plain text readme",
			},
			expectFileName: "README",
			expectContent:  "plain text readme",
		},
		{
			name:  "NoReadme",
			files: map[string]string{},
			// expect error
		},
	}

	for _, tc := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if content, ok := tc.files[r.URL.Path]; ok {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(content))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		cfg := &config.BitbucketConfig{Workspace: "ws"}
		c := NewClient(cfg)
		c.baseURL = server.URL // override for test

		name, content, err := c.GetRepositoryReadme("repo")
		if tc.expectFileName == "" { // expecting error
			if err == nil {
				server.Close()
				// fail if no error when none expected file
				t.Fatalf("%s: expected error for missing README", tc.name)
			}
			continue
		}
		if err != nil {
			server.Close()
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if name != tc.expectFileName {
			server.Close()
			t.Fatalf("%s: expected filename %s, got %s", tc.name, tc.expectFileName, name)
		}
		if content != tc.expectContent {
			server.Close()
			t.Fatalf("%s: expected content %q, got %q", tc.name, tc.expectContent, content)
		}
	}
}
