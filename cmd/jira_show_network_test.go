package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"devflow/internal/config"
	"devflow/internal/httpx"
	"devflow/internal/jira"
)

func TestJiraChildAndRecursiveHelpers(t *testing.T) {
	server := httpx.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/3/search/jql":
			jql := r.URL.Query().Get("jql")
			switch {
			case strings.Contains(jql, "parent = ROOT-1"):
				_ = json.NewEncoder(w).Encode(jira.SearchResponse{Issues: []jira.Issue{{ID: "2", Key: "CHILD-1"}}})
			case strings.Contains(jql, "parent = CHILD-1"):
				_ = json.NewEncoder(w).Encode(jira.SearchResponse{Issues: []jira.Issue{{ID: "3", Key: "GRAND-1"}}})
			default:
				_ = json.NewEncoder(w).Encode(jira.SearchResponse{Issues: []jira.Issue{}})
			}
		case "/rest/dev-status/1.0/issue/detail":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"detail": []any{map[string]any{
					"pullRequests": []map[string]any{{"id": "pr-1", "name": "Implement child"}},
				}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := jira.NewClient(&config.JiraConfig{URL: server.URL})

	children, err := fetchChildIssues(client, "ROOT-1")
	if err != nil || len(children) != 1 || children[0].Key != "CHILD-1" {
		t.Fatalf("fetchChildIssues() = %+v, err=%v", children, err)
	}

	tree, err := buildIssueTree(client, "ROOT-1", true)
	if err != nil {
		t.Fatalf("buildIssueTree() failed: %v", err)
	}
	if len(tree) != 1 || len(tree[0].Children) != 1 || len(tree[0].PullRequests) != 1 {
		t.Fatalf("unexpected issue tree: %+v", tree)
	}

	root := &jira.IssueDetails{ID: "1", Key: "ROOT-1"}
	root.Fields.Summary = "Root"
	out := captureStdout(func() {
		displayPullRequests(client, root)
	})
	if !strings.Contains(out, "Implement child") {
		t.Fatalf("pull-request display missing result: %s", out)
	}

	cfg := &config.Config{Jira: config.JiraConfig{URL: server.URL}}
	out = captureStdout(func() {
		displayChildIssues(client, cfg, "ROOT-1")
	})
	if !strings.Contains(out, "CHILD-1") || !strings.Contains(out, "Child") {
		t.Fatalf("child display missing result: %s", out)
	}
}
