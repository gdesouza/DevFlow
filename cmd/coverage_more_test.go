package cmd

import (
	"bufio"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"devflow/internal/jira"
)

type fakeRepositoryPager struct {
	repos      []bitbucket.Repository
	totalCount int
	pageCount  int
	lastPage   int
	lastSize   int
}

func (f *fakeRepositoryPager) GetRepositoriesPaged(page, size int) ([]bitbucket.Repository, int, error) {
	f.pageCount++
	f.lastPage = page
	f.lastSize = size
	return f.repos, f.totalCount, nil
}

type fakePipelineLister struct {
	pipelines []bitbucket.Pipeline
	err       error
	limitSeen int
}

func (f *fakePipelineLister) GetPipelines(repoSlug string, limit int) ([]bitbucket.Pipeline, error) {
	f.limitSeen = limit
	return f.pipelines, f.err
}

func TestDeriveSlug(t *testing.T) {
	cases := map[string]string{
		"Repo Name":            "repo-name",
		"  Mixed   CASE  Name": "mixed-case-name",
		"Already-Slugged":      "already-slugged",
	}

	for in, want := range cases {
		if got := deriveSlug(in); got != want {
			t.Fatalf("deriveSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGetConfigValueAndSetConfigValue(t *testing.T) {
	cfg := &config.Config{}

	if err := setConfigValue(cfg, "jira.url", "https://jira.example"); err != nil {
		t.Fatalf("setConfigValue jira.url failed: %v", err)
	}
	if err := setConfigValue(cfg, "bitbucket.workspace", "workspace"); err != nil {
		t.Fatalf("setConfigValue bitbucket.workspace failed: %v", err)
	}
	if err := setConfigValue(cfg, "bitbucket.bitbucket_user", "bb-user"); err != nil {
		t.Fatalf("setConfigValue bitbucket.bitbucket_user failed: %v", err)
	}

	cases := map[string]string{
		"jira.url":                 "https://jira.example",
		"bitbucket.workspace":      "workspace",
		"bitbucket.bitbucket_user": "bb-user",
	}

	for key, want := range cases {
		got, err := getConfigValue(cfg, key)
		if err != nil {
			t.Fatalf("getConfigValue(%q) returned error: %v", key, err)
		}
		if got != want {
			t.Fatalf("getConfigValue(%q) = %q, want %q", key, got, want)
		}
	}

	if _, err := getConfigValue(cfg, "badformat"); err == nil {
		t.Fatal("expected error for invalid key format")
	}
	if _, err := getConfigValue(cfg, "unknown.field"); err == nil {
		t.Fatal("expected error for unknown section")
	}
}

func TestPromptWithDefault(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	if got := promptWithDefault(reader, "Prompt", "default"); got != "default" {
		t.Fatalf("promptWithDefault default = %q, want %q", got, "default")
	}

	reader = bufio.NewReader(strings.NewReader("custom\n"))
	if got := promptWithDefault(reader, "Prompt", "default"); got != "custom" {
		t.Fatalf("promptWithDefault custom = %q, want %q", got, "custom")
	}
}

func TestSetToSortedSliceAndPrintWatchedSummary(t *testing.T) {
	set := map[string]struct{}{"z": {}, "a": {}, "m": {}}
	got := setToSortedSlice(set)
	want := []string{"a", "m", "z"}
	if len(got) != len(want) {
		t.Fatalf("setToSortedSlice len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("setToSortedSlice[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	out := captureStdout(func() {
		printWatchedSummary(nil)
	})
	if !strings.Contains(out, "No watched repositories.") {
		t.Fatalf("unexpected empty summary output: %q", out)
	}

	out = captureStdout(func() {
		printWatchedSummary([]string{"a", "b"})
	})
	if !strings.Contains(out, "Watched (2): a, b") {
		t.Fatalf("unexpected watched summary output: %q", out)
	}
}

func TestRunPagedMode(t *testing.T) {
	pager := &fakeRepositoryPager{
		repos: []bitbucket.Repository{
			{Name: "repo-a", FullName: "ws/repo-a", Language: "Go"},
		},
		totalCount: 1,
	}

	out := captureStdout(func() {
		runPagedMode(pager, "workspace", 0, 20)
	})
	if pager.pageCount != 1 || pager.lastPage != 0 || pager.lastSize != 20 {
		t.Fatalf("unexpected pager calls: %+v", pager)
	}
	if !strings.Contains(out, "Found 1 repositories in workspace 'workspace'") {
		t.Fatalf("unexpected runPagedMode output: %q", out)
	}
}

func TestRunPagedMode_Empty(t *testing.T) {
	pager := &fakeRepositoryPager{}
	out := captureStdout(func() {
		runPagedMode(pager, "workspace", 0, 20)
	})
	if !strings.Contains(out, "No repositories found in workspace 'workspace'.") {
		t.Fatalf("unexpected empty paging output: %q", out)
	}
}

func TestResolvePipelineUUID(t *testing.T) {
	lister := &fakePipelineLister{
		pipelines: []bitbucket.Pipeline{{UUID: "{abc-123}", BuildNumber: 42}},
	}

	got, err := resolvePipelineUUID(lister, "repo", "42")
	if err != nil {
		t.Fatalf("resolvePipelineUUID failed: %v", err)
	}
	if got != "{abc-123}" {
		t.Fatalf("resolvePipelineUUID build number = %q", got)
	}
	if lister.limitSeen != 100 {
		t.Fatalf("expected limit 100, got %d", lister.limitSeen)
	}

	if got, err := resolvePipelineUUID(lister, "repo", "{direct-uuid}"); err != nil || got != "{direct-uuid}" {
		t.Fatalf("resolvePipelineUUID direct uuid = %q, err=%v", got, err)
	}

	if _, err := resolvePipelineUUID(lister, "repo", "999"); err == nil || !strings.Contains(err.Error(), "no pipeline found") {
		t.Fatalf("expected not-found error, got %v", err)
	}
}

func TestPipelineHelpers(t *testing.T) {
	if got := countRune("a-b-c-d", '-'); got != 3 {
		t.Fatalf("countRune = %d, want 3", got)
	}
	if got := formatSeconds(45); got != "45s" {
		t.Fatalf("formatSeconds(45) = %q", got)
	}
	if got := formatSeconds(120); got != "2m" {
		t.Fatalf("formatSeconds(120) = %q", got)
	}
	if got := formatSeconds(125); got != "2m5s" {
		t.Fatalf("formatSeconds(125) = %q", got)
	}

	pipeline := bitbucket.Pipeline{}
	pipeline.State.Name = "COMPLETED"
	pipeline.State.Result = &struct {
		Name string `json:"name"`
	}{Name: "SUCCESSFUL"}
	if got := pipelineStateLabel(pipeline); got != "✅ SUCCESSFUL" {
		t.Fatalf("pipelineStateLabel completed = %q", got)
	}

	pipeline = bitbucket.Pipeline{}
	pipeline.State.Name = "IN_PROGRESS"
	pipeline.State.Stage = &struct {
		Name string `json:"name"`
	}{Name: "Build"}
	if got := pipelineStateLabel(pipeline); got != "🔄 Build" {
		t.Fatalf("pipelineStateLabel progress = %q", got)
	}

	cases := map[string]string{
		"SUCCESSFUL": "✅",
		"FAILED":     "❌",
		"ERROR":      "💥",
		"STOPPED":    "🛑",
		"OTHER":      "📝",
	}
	for in, want := range cases {
		if got := pipelineResultIcon(in); got != want {
			t.Fatalf("pipelineResultIcon(%q) = %q, want %q", in, got, want)
		}
	}

	step := bitbucket.PipelineStep{}
	step.State.Name = "COMPLETED"
	step.State.Result = &struct {
		Name string `json:"name"`
	}{Name: "FAILED"}
	if got := stepStateLabel(step); got != "❌ FAILED" {
		t.Fatalf("stepStateLabel completed = %q", got)
	}

	step.State.Name = "IN_PROGRESS"
	if got := stepStateLabel(step); got != "🔄 RUNNING" {
		t.Fatalf("stepStateLabel progress = %q", got)
	}

	pipeline.BuildSecondsUsed = 0
	if got := pipelineDuration(pipeline); got != "-" {
		t.Fatalf("pipelineDuration zero = %q", got)
	}
	pipeline.BuildSecondsUsed = 3661
	if got := pipelineDuration(pipeline); got != "61m1s" {
		t.Fatalf("pipelineDuration 3661 = %q", got)
	}

	step.DurationInSeconds = 59
	if got := stepDuration(step); got != "59s" {
		t.Fatalf("stepDuration 59 = %q", got)
	}
	step.DurationInSeconds = 3600
	if got := stepDuration(step); got != "60m" {
		t.Fatalf("stepDuration 3600 = %q", got)
	}
}

func TestExtractRepoSlugFromPRAndDisplayPRsToReview(t *testing.T) {
	pr := bitbucket.PullRequestWithReviewers{}
	pr.Source.Repository.Name = "source-repo"
	if got := extractRepoSlugFromPR(pr); got != "source-repo" {
		t.Fatalf("extractRepoSlugFromPR source = %q", got)
	}
	pr.Source.Repository.Name = ""
	pr.Destination.Repository.Name = "dest-repo"
	if got := extractRepoSlugFromPR(pr); got != "dest-repo" {
		t.Fatalf("extractRepoSlugFromPR dest = %q", got)
	}

	out := captureStdout(func() {
		displayPRsToReview([]PRWithRepo{
			{
				RepoSlug: "repo",
				PR: bitbucket.PullRequestWithReviewers{
					ID:    1,
					Title: "Example PR",
					State: "OPEN",
					Author: struct {
						DisplayName string `json:"display_name"`
					}{DisplayName: "Alice"},
					Source: struct {
						Branch struct {
							Name string `json:"name"`
						} `json:"branch"`
						Repository struct {
							Name string `json:"name"`
						} `json:"repository"`
					}{Branch: struct {
						Name string `json:"name"`
					}{Name: "feature"}, Repository: struct {
						Name string `json:"name"`
					}{Name: "repo"}},
					Destination: struct {
						Branch struct {
							Name string `json:"name"`
						} `json:"branch"`
						Repository struct {
							Name string `json:"name"`
						} `json:"repository"`
					}{Branch: struct {
						Name string `json:"name"`
					}{Name: "main"}, Repository: struct {
						Name string `json:"name"`
					}{Name: "repo"}},
				},
			},
		}, "source", true, true, "workspace")
	})

	var jsonOut struct {
		Workspace string `json:"workspace"`
		Source    string `json:"source"`
		TotalPRs  int    `json:"total_prs"`
	}
	if err := json.Unmarshal([]byte(out), &jsonOut); err != nil {
		t.Fatalf("failed to unmarshal displayPRsToReview json: %v", err)
	}
	if jsonOut.Workspace != "workspace" || jsonOut.Source != "source" || jsonOut.TotalPRs != 1 {
		t.Fatalf("unexpected JSON output: %+v", jsonOut)
	}
}

func TestFilterPRsForUserAndDisplayText(t *testing.T) {
	prs := []bitbucket.PullRequestWithReviewers{
		{
			ID:    1,
			Title: "Match Me",
			Reviewers: []struct {
				DisplayName string `json:"display_name"`
				UUID        string `json:"uuid"`
			}{{DisplayName: "Alice"}},
		},
		{
			ID:    2,
			Title: "Ignore Me",
			Reviewers: []struct {
				DisplayName string `json:"display_name"`
				UUID        string `json:"uuid"`
			}{{DisplayName: "Bob"}},
		},
	}

	filtered := filterPRsForUser(prs, "alice", true)
	if len(filtered) != 1 || filtered[0].ID != 1 {
		t.Fatalf("unexpected filtered PRs: %+v", filtered)
	}

	textOut := captureStdout(func() {
		displayPRsToReview([]PRWithRepo{}, "watched repos", false, false, "workspace")
	})
	if !strings.Contains(textOut, "No pull requests found") {
		t.Fatalf("unexpected text PR output: %q", textOut)
	}
}

func TestDisplayReposPageAndIssueDetails(t *testing.T) {
	repos := []bitbucket.Repository{
		{Name: "repo-a", FullName: "ws/repo-a", IsPrivate: true, Language: "Go"},
		{Name: "repo-b", FullName: "ws/repo-b", IsPrivate: false},
	}
	reposOut := captureStdout(func() {
		displayReposPage(repos, "workspace")
	})
	if !strings.Contains(reposOut, "repo-a") || !strings.Contains(reposOut, "Legend:") {
		t.Fatalf("unexpected repos output: %q", reposOut)
	}

	issue := &jira.IssueDetails{Key: "ABC-1"}
	issue.Fields.Summary = "Test issue"
	issue.Fields.Status.Name = "In Progress"
	issue.Fields.Priority.Name = "High"
	issue.Fields.Assignee.DisplayName = "Alice"
	issue.Fields.Reporter.DisplayName = "Bob"
	issue.Fields.Description = map[string]any{"type": "doc", "content": []any{map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "Hello"}}}}}
	issueOut := captureStdout(func() {
		displayIssueDetails(issue)
	})
	if !strings.Contains(issueOut, "ABC-1") || !strings.Contains(issueOut, "Test issue") || !strings.Contains(issueOut, "Description") {
		t.Fatalf("unexpected issue output: %q", issueOut)
	}
}

func TestFormatSizeAndRelativeTime(t *testing.T) {
	if got := formatSize(512); got != "512 B" {
		t.Fatalf("formatSize(512) = %q", got)
	}
	if got := formatSize(1024 * 1024); got != "1.0 MB" {
		t.Fatalf("formatSize(1MB) = %q", got)
	}

	if got := formatRelativeTime(time.Now().Add(-10 * time.Second)); got != "just now" {
		t.Fatalf("formatRelativeTime just now = %q", got)
	}
	if got := formatRelativeTime(time.Now().Add(-2 * time.Hour)); got != "2 hours ago" {
		t.Fatalf("formatRelativeTime hours = %q", got)
	}
	if got := formatRelativeTime(time.Now().Add(-48 * time.Hour)); got != "2 days ago" {
		t.Fatalf("formatRelativeTime days = %q", got)
	}
}
