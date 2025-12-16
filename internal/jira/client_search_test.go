package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"devflow/internal/config"
)

func TestSearch_FreeTextAndJQL(t *testing.T) {
	cases := []struct {
		name       string
		isJQL      bool
		query      string
		maxResults int
		expectJQL  string
	}{
		{
			name:       "free-text",
			isJQL:      false,
			query:      "server crash",
			maxResults: 0,
			expectJQL:  `text ~ "server crash" ORDER BY updated DESC`,
		},
		{
			name:       "raw-jql",
			isJQL:      true,
			query:      `project = ABC AND status = "To Do"`,
			maxResults: 10,
			expectJQL:  `project = ABC AND status = "To Do" ORDER BY updated DESC`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server that validates received query params
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/rest/api/3/search/jql" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}

				vals := r.URL.Query()
				jql := vals.Get("jql")
				if jql != tc.expectJQL {
					t.Fatalf("unexpected jql: got=%q want=%q", jql, tc.expectJQL)
				}
				if tc.maxResults > 0 {
					mr := vals.Get("maxResults")
					if mr != fmt.Sprintf("%d", tc.maxResults) {
						t.Fatalf("unexpected maxResults: got=%q want=%d", mr, tc.maxResults)
					}
				}

				// Return a simple search response as JSON
				respJSON := `{"issues":[{"key":"ABC-1","fields":{"summary":"Test issue","status":{"name":"To Do"},"assignee":{"displayName":"Alice"},"priority":{"name":"High"}}}]}`
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(respJSON)); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))
			defer srv.Close()

			// Prepare client
			cfg := &config.JiraConfig{
				URL:      srv.URL,
				Username: "me",
				Token:    "token",
			}
			c := NewClient(cfg)

			// Call Search
			issues, err := c.Search(tc.query, tc.isJQL, tc.maxResults, 0)
			if err != nil {
				t.Fatalf("Search returned error: %v", err)
			}
			if len(issues) != 1 {
				t.Fatalf("expected 1 issue, got %d", len(issues))
			}
			if issues[0].Key != "ABC-1" {
				t.Fatalf("unexpected issue key: %s", issues[0].Key)
			}
		})
	}
}

func TestGetMyIssuesAndFindMentions(t *testing.T) {
	// This server will validate the JQL for GetMyIssues and FindMentions
	calls := make([]string, 0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		vals := r.URL.Query()
		jql := vals.Get("jql")
		calls = append(calls, jql)

		resp := SearchResponse{Issues: []Issue{}}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(b); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer srv.Close()

	cfg := &config.JiraConfig{URL: srv.URL, Username: "me_user", Token: "t"}
	c := NewClient(cfg)

	// GetMyIssues should call assignee = currentUser()
	_, err := c.GetMyIssues()
	if err != nil {
		t.Fatalf("GetMyIssues error: %v", err)
	}
	if len(calls) < 1 {
		t.Fatalf("expected at least one call, got 0")
	}
	if calls[0] != "assignee = currentUser() ORDER BY updated DESC" {
		t.Fatalf("unexpected GetMyIssues jql: %q", calls[0])
	}

	// Reset and call FindMentions
	calls = calls[:0]
	_, err = c.FindMentions()
	if err != nil {
		t.Fatalf("FindMentions error: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call to FindMentions, got %d", len(calls))
	}
	// The username should be embedded in the jql
	expected := fmt.Sprintf("text ~ \"%s\" ORDER BY updated DESC", cfg.Username)
	if calls[0] != expected {
		t.Fatalf("unexpected FindMentions jql: got=%q want=%q", calls[0], expected)
	}
}

func TestSearch_MultiPagePaging(t *testing.T) {
	// Simulate a server that returns 2 issues per page, total 3 issues
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		vals := r.URL.Query()
		startAt := 0
		if sa := vals.Get("startAt"); sa != "" {
			fmt.Sscanf(sa, "%d", &startAt)
		}
		maxResults := 2
		if mr := vals.Get("maxResults"); mr != "" {
			fmt.Sscanf(mr, "%d", &maxResults)
		}

		var resp SearchResponse
		resp.StartAt = startAt
		resp.MaxResults = maxResults
		resp.Total = 3
		if startAt == 0 {
			resp.Issues = []Issue{{Key: "ABC-1"}, {Key: "ABC-2"}}
		} else if startAt == 2 {
			resp.Issues = []Issue{{Key: "ABC-3"}}
		} else {
			resp.Issues = []Issue{}
		}

		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(b); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer srv.Close()

	cfg := &config.JiraConfig{URL: srv.URL, Username: "u", Token: "t"}
	c := NewClient(cfg)

	issues, err := c.Search("project = ABC", true, 3, 0)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}
	if issues[0].Key != "ABC-1" || issues[1].Key != "ABC-2" || issues[2].Key != "ABC-3" {
		t.Fatalf("unexpected issue keys: %v", []string{issues[0].Key, issues[1].Key, issues[2].Key})
	}
	if calls < 2 {
		t.Fatalf("expected multiple calls to fetch pages, got %d", calls)
	}
}

func TestSearch_EncodingsAndQueryEscape(t *testing.T) {
	// Ensure special characters are preserved and decoded by the server
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.RawQuery
		// Return minimal response
		resp := SearchResponse{Issues: []Issue{}}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(b); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer srv.Close()

	cfg := &config.JiraConfig{URL: srv.URL, Username: "u", Token: "t"}
	c := NewClient(cfg)

	query := `summary ~ "needs & review"` // contains spaces and ampersand
	_, err := c.Search(query, true, 0, 0)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	// raw query should contain jql=... percent-encoded
	if seen == "" {
		t.Fatalf("server did not receive query string")
	}
	vals, _ := url.ParseQuery(seen)
	if vals.Get("jql") == "" {
		t.Fatalf("server did not receive jql param in RawQuery: %s", seen)
	}
}

func TestSearch_FallbackCap(t *testing.T) {
	// Server that omits paging metadata but returns some issues
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		// Return issues but zeroed paging metadata
		resp := SearchResponse{Issues: []Issue{{Key: "X-1"}, {Key: "X-2"}, {Key: "X-3"}}, StartAt: 0, MaxResults: 0, Total: 0}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(b); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer srv.Close()

	cfg := &config.JiraConfig{URL: srv.URL, Username: "u", Token: "t"}
	c := NewClient(cfg)

	// Request a page that would require fallbackSize > cap (cap = 500 in client)
	_, err := c.Search("project = X", true, 100, 1000) // startAtArg=1000, perPage=100 => fallbackSize=1100
	if err == nil {
		t.Fatalf("expected error due to fallback cap, got nil")
	}
}
