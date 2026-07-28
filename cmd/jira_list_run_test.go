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

func TestJiraListCmd(t *testing.T) {
	origLoad := loadConfig
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Jira: config.JiraConfig{
				URL:      "https://jira.example",
				Username: "alice",
				Token:    "token",
			},
		}, nil
	}
	t.Cleanup(func() { loadConfig = origLoad })

	httpx.RegisterTestServer("jira.example", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/rest/api/3/search/jql") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		makeIssue := func(key, summary, status, priority string) jira.Issue {
			var issue jira.Issue
			issue.Key = key
			issue.Fields.Summary = summary
			issue.Fields.Status.Name = status
			issue.Fields.Priority.Name = priority
			issue.Fields.Assignee.DisplayName = "Alice"
			issue.Fields.Sprint = map[string]any{"name": "Sprint 1"}
			return issue
		}
		var resp jira.SearchResponse
		switch {
		case strings.Contains(r.URL.RawQuery, "api"):
			resp = jira.SearchResponse{
				Issues: []jira.Issue{
					makeIssue("ABC-2", "API work", "In Progress", "High"),
					makeIssue("ABC-1", "Backend work", "Done", "Low"),
				},
			}
		default:
			resp = jira.SearchResponse{
				Issues: []jira.Issue{
					makeIssue("ABC-3", "My issue", "Open", "Medium"),
				},
			}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(func() { httpx.UnregisterTestServer("jira.example") })

	origFilter := filterStatus
	origSort := sortBy
	origPriority := showPriority
	origSprint := showSprint
	origExclude := excludeDone
	origQuery := searchQuery
	origJQL := searchJQL
	origMax := maxResults
	origPage := page
	origFetch := fetchAll
	defer func() {
		filterStatus = origFilter
		sortBy = origSort
		showPriority = origPriority
		showSprint = origSprint
		excludeDone = origExclude
		searchQuery = origQuery
		searchJQL = origJQL
		maxResults = origMax
		page = origPage
		fetchAll = origFetch
	}()

	filterStatus = ""
	sortBy = "status"
	showPriority = false
	showSprint = false
	excludeDone = false
	searchQuery = ""
	searchJQL = ""
	maxResults = 0
	page = 0
	fetchAll = false

	out := captureStdout(func() {
		listTasksCmd.Run(listTasksCmd, nil)
	})
	if !strings.Contains(out, "Found 1 Jira tasks assigned to you") || !strings.Contains(out, "ABC-3") {
		t.Fatalf("unexpected default jira list output: %q", out)
	}

	searchQuery = "api"
	maxResults = 10
	showPriority = true
	showSprint = true
	filterStatus = "In Progress"
	excludeDone = true
	sortBy = "priority"
	fetchAll = true
	out = captureStdout(func() {
		listTasksCmd.Run(listTasksCmd, nil)
	})
	if !strings.Contains(out, "Found 1 Jira tasks assigned to you (filtered by status: In Progress) (excluding done tasks)") {
		t.Fatalf("unexpected filtered jira list output: %q", out)
	}
	if !strings.Contains(out, "ABC-2") || !strings.Contains(out, "Sprint:") {
		t.Fatalf("unexpected filtered jira list body: %q", out)
	}
}
