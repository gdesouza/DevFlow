package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"devflow/internal/bitbucket"
)

func makeTestPR(id int, title string, reviewers []string) bitbucket.PullRequestWithReviewers {
	pr := bitbucket.PullRequestWithReviewers{
		ID:    id,
		Title: title,
		Author: struct {
			DisplayName string `json:"display_name"`
		}{DisplayName: reviewers[0]}, // Use first reviewer as author for test simplicity
	}
	for _, name := range reviewers {
		pr.Reviewers = append(pr.Reviewers, struct {
			DisplayName string `json:"display_name"`
			UUID        string `json:"uuid"`
		}{DisplayName: name, UUID: "uidx"})
	}
	return pr
}

func TestFilterPRsForUser(t *testing.T) {
	cases := []struct {
		name   string
		prs    []bitbucket.PullRequestWithReviewers
		user   string
		expect []int // expected IDs
	}{
		{
			name:   "empty list",
			prs:    []bitbucket.PullRequestWithReviewers{},
			user:   "bob",
			expect: []int{},
		},
		{
			name: "no reviewer match",
			prs: []bitbucket.PullRequestWithReviewers{
				makeTestPR(1, "Test One", []string{"alice"}),
				makeTestPR(2, "Test Two", []string{"carol"}),
			},
			user:   "bob",
			expect: []int{},
		},
		{
			name: "single match",
			prs: []bitbucket.PullRequestWithReviewers{
				makeTestPR(1, "Test One", []string{"bob"}),
				makeTestPR(2, "Test Two", []string{"alice"}),
			},
			user:   "bob",
			expect: []int{1},
		},
		{
			name: "case insensitive match",
			prs: []bitbucket.PullRequestWithReviewers{
				makeTestPR(5, "Case PR", []string{"Bob"}),
			},
			user:   "bob",
			expect: []int{5},
		},
		{
			name: "multiple matches",
			prs: []bitbucket.PullRequestWithReviewers{
				makeTestPR(3, "A", []string{"bob"}),
				makeTestPR(4, "B", []string{"bob", "alice"}),
			},
			user:   "bob",
			expect: []int{3, 4},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := filterPRsForUser(tc.prs, tc.user, true)
			var got []int
			for _, pr := range result {
				got = append(got, pr.ID)
			}
			if len(got) != len(tc.expect) {
				t.Fatalf("expected %v, got %v", tc.expect, got)
			}
			for i := range got {
				if got[i] != tc.expect[i] {
					t.Errorf("mismatch: want %v got %v", tc.expect, got)
				}
			}
		})
	}
}

// Minimal smoke test for displayPRsToReview JSON output
type fakePRWithRepo struct {
	PR       bitbucket.PullRequestWithReviewers
	RepoSlug string
}

func TestDisplayPRsToReview_EmptyText(t *testing.T) {
	var prs []PRWithRepo
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()
	displayPRsToReview(prs, "mySrc", false, false, "myworkspace")
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No pull requests found") {
		t.Errorf("expected message for empty PR list, got: %q", buf.String())
	}
}

func TestDisplayPRsToReview_TextMultiple(t *testing.T) {
	prs := []PRWithRepo{
		{PR: makeTestPR(1, "Title 1", []string{"alice"}), RepoSlug: "repo1"},
		{PR: makeTestPR(2, "Title 2", []string{"bob"}), RepoSlug: "repo2"},
	}
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()
	displayPRsToReview(prs, "src", true, false, "myworkspace")
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	for _, pr := range prs {
		if !strings.Contains(out, pr.PR.Title) {
			t.Errorf("missing PR title %q in output: %s", pr.PR.Title, out)
		}
		if !strings.Contains(out, pr.RepoSlug) {
			t.Errorf("missing repo slug %q in output: %s", pr.RepoSlug, out)
		}
		if !strings.Contains(out, "Author: "+pr.PR.Author.DisplayName) {
			t.Errorf("missing author for PR: %d", pr.PR.ID)
		}
	}
}

func TestDisplayPRsToReview_JSON(t *testing.T) {
	pr := makeTestPR(101, "Review me!", []string{"bob"})
	prs := []PRWithRepo{{PR: pr, RepoSlug: "myrepo"}}

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()
	displayPRsToReview(prs, "src", true, true, "myworkspace")
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	var out struct {
		Workspace    string       `json:"workspace"`
		Source       string       `json:"source"`
		TotalPRs     int          `json:"total_prs"`
		PullRequests []PRWithRepo `json:"pull_requests"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("Failed parsing output: %v", err)
	}
	if out.TotalPRs != 1 {
		t.Errorf("Bad output: %+v", out)
	}
}
