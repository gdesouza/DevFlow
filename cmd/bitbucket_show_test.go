package cmd

import (
	"io"
	"os"
	"strings"
	"testing"

	"devflow/internal/bitbucket"
)

func TestGetPRStatusIcon(t *testing.T) {
	cases := []struct {
		state string
		want  string
	}{
		{"OPEN", "🔄"},
		{"MERGED", "✅"},
		{"DECLINED", "❌"},
		{"REJECTED", "❌"},
		{"SUPERSEDED", "🔄"},
		{"UNKNOWN_STATE", "📝"},
		{"", "📝"},
	}
	for _, tc := range cases {
		got := getPRStatusIcon(tc.state)
		if got != tc.want {
			t.Errorf("getPRStatusIcon(%q) = %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestDisplayPRDetails(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	pr := &bitbucket.PullRequestDetails{
		ID:          42,
		Title:       "Test Pull Request",
		State:       "OPEN",
		Description: "This is a test description",
		CreatedOn:   "2025-01-01T12:00:00+00:00",
		UpdatedOn:   "2025-01-02T12:00:00+00:00",
	}
	pr.Author.DisplayName = "Alice"
	pr.Source.Branch.Name = "feature-branch"
	pr.Destination.Branch.Name = "main"
	pr.Reviewers = []struct {
		DisplayName string `json:"display_name"`
	}{
		{DisplayName: "Bob"},
		{DisplayName: "Charlie"},
	}

	displayPRDetails(pr, "my-workspace", "my-repo")

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	output := string(out)

	// Verify key elements are present
	if !strings.Contains(output, "#42") {
		t.Errorf("expected output to contain PR ID #42")
	}
	if !strings.Contains(output, "Test Pull Request") {
		t.Errorf("expected output to contain PR title")
	}
	if !strings.Contains(output, "https://bitbucket.org/my-workspace/my-repo/pull-requests/42") {
		t.Errorf("expected output to contain PR URL")
	}
	if !strings.Contains(output, "OPEN") {
		t.Errorf("expected output to contain state OPEN")
	}
	if !strings.Contains(output, "Alice") {
		t.Errorf("expected output to contain author Alice")
	}
	if !strings.Contains(output, "feature-branch") {
		t.Errorf("expected output to contain source branch")
	}
	if !strings.Contains(output, "main") {
		t.Errorf("expected output to contain destination branch")
	}
	if !strings.Contains(output, "This is a test description") {
		t.Errorf("expected output to contain description")
	}
	if !strings.Contains(output, "Bob") {
		t.Errorf("expected output to contain reviewer Bob")
	}
	if !strings.Contains(output, "Charlie") {
		t.Errorf("expected output to contain reviewer Charlie")
	}
	if !strings.Contains(output, "2025-01-01") {
		t.Errorf("expected output to contain created date")
	}
	if !strings.Contains(output, "2025-01-02") {
		t.Errorf("expected output to contain updated date")
	}
}

func TestDisplayPRDetails_NoDescription(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	pr := &bitbucket.PullRequestDetails{
		ID:          1,
		Title:       "No Description PR",
		State:       "MERGED",
		Description: "",
	}
	pr.Author.DisplayName = "Dev"
	pr.Source.Branch.Name = "hotfix"
	pr.Destination.Branch.Name = "master"

	displayPRDetails(pr, "ws", "repo")

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	output := string(out)

	// Should not contain description section header when description is empty
	if strings.Contains(output, "📄 Description:") {
		t.Errorf("expected no description section when description is empty")
	}
}

func TestDisplayPRDetails_NoReviewers(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	pr := &bitbucket.PullRequestDetails{
		ID:    2,
		Title: "No Reviewers PR",
		State: "DECLINED",
	}
	pr.Author.DisplayName = "Dev"
	pr.Source.Branch.Name = "feature"
	pr.Destination.Branch.Name = "main"
	pr.Reviewers = nil

	displayPRDetails(pr, "ws", "repo")

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	output := string(out)

	// Should not contain reviewers section when no reviewers
	if strings.Contains(output, "👥 Reviewers") {
		t.Errorf("expected no reviewers section when reviewers list is empty")
	}
}

func TestDisplayPRDetails_NoDates(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	pr := &bitbucket.PullRequestDetails{
		ID:        3,
		Title:     "No Dates PR",
		State:     "OPEN",
		CreatedOn: "",
		UpdatedOn: "",
	}
	pr.Author.DisplayName = "Dev"
	pr.Source.Branch.Name = "feature"
	pr.Destination.Branch.Name = "main"

	displayPRDetails(pr, "ws", "repo")

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	output := string(out)

	// Should not contain date lines when dates are empty
	if strings.Contains(output, "📅 Created:") {
		t.Errorf("expected no created date section when CreatedOn is empty")
	}
	if strings.Contains(output, "🔄 Updated:") {
		t.Errorf("expected no updated date section when UpdatedOn is empty")
	}
}

func TestDisplayPRDetails_AllStates(t *testing.T) {
	states := []string{"OPEN", "MERGED", "DECLINED", "SUPERSEDED", "UNKNOWN"}

	for _, state := range states {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		pr := &bitbucket.PullRequestDetails{
			ID:    1,
			Title: "Test PR",
			State: state,
		}
		pr.Author.DisplayName = "Dev"
		pr.Source.Branch.Name = "feature"
		pr.Destination.Branch.Name = "main"

		displayPRDetails(pr, "ws", "repo")

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = oldStdout

		output := string(out)
		if !strings.Contains(output, state) {
			t.Errorf("expected output to contain state %s", state)
		}
	}
}

func TestPrintRepoPRs(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	prs := []bitbucket.PullRequest{
		{ID: 1, Title: "First PR", State: "OPEN"},
		{ID: 2, Title: "Second PR", State: "MERGED"},
	}

	printRepoPRs("workspace", "my-repo", prs)

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	output := string(out)

	if !strings.Contains(output, "Repository: my-repo (2 PRs)") {
		t.Errorf("expected repository header with PR count")
	}
	if !strings.Contains(output, "#1 - First PR") {
		t.Errorf("expected first PR")
	}
	if !strings.Contains(output, "#2 - Second PR") {
		t.Errorf("expected second PR")
	}
	if !strings.Contains(output, "https://bitbucket.org/workspace/my-repo/pull-requests/1") {
		t.Errorf("expected PR URL")
	}
}

func TestPrintRepoPRs_Empty(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRepoPRs("workspace", "my-repo", []bitbucket.PullRequest{})

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	output := string(out)
	if output != "" {
		t.Errorf("expected no output for empty PR list, got: %s", output)
	}
}

func TestShowPRCmd_Initialization(t *testing.T) {
	// Verify the command is properly configured
	if showPRCmd.Use != "show [repo-slug] [pr-id]" {
		t.Errorf("unexpected Use: %s", showPRCmd.Use)
	}
	if len(showPRCmd.Aliases) != 1 || showPRCmd.Aliases[0] != "show-pr" {
		t.Errorf("unexpected aliases: %v", showPRCmd.Aliases)
	}

	// Verify --diff flag exists
	flag := showPRCmd.Flags().Lookup("diff")
	if flag == nil {
		t.Errorf("expected --diff flag to be defined")
	}
	if flag != nil && flag.DefValue != "false" {
		t.Errorf("expected --diff default to be false, got %s", flag.DefValue)
	}
}

func TestListPRsCmd_Initialization(t *testing.T) {
	// Verify the command is properly configured
	if listPRsCmd.Use != "list [repo-slug]" {
		t.Errorf("unexpected Use: %s", listPRsCmd.Use)
	}
	if len(listPRsCmd.Aliases) != 1 || listPRsCmd.Aliases[0] != "list-prs" {
		t.Errorf("unexpected aliases: %v", listPRsCmd.Aliases)
	}

	// Verify --repo flag exists
	flag := listPRsCmd.Flags().Lookup("repo")
	if flag == nil {
		t.Errorf("expected --repo flag to be defined")
	}
}
