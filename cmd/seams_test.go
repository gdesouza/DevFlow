package cmd

import (
	"os"
	"strings"
	"testing"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"devflow/internal/jira"
	"golang.org/x/term"
)

func TestSaveWatched(t *testing.T) {
	origLoad := loadConfig
	origSave := saveConfig
	defer func() {
		loadConfig = origLoad
		saveConfig = origSave
	}()

	var saved *config.Config
	loadConfig = func() (*config.Config, error) {
		return &config.Config{}, nil
	}
	saveConfig = func(cfg *config.Config) error {
		saved = cfg
		return nil
	}

	saveWatched(map[string]struct{}{"z": {}, "a": {}})
	if saved == nil || len(saved.Bitbucket.WatchedRepos) != 2 {
		t.Fatalf("expected watched repos to be saved, got %+v", saved)
	}
	if saved.Bitbucket.WatchedRepos[0] != "a" || saved.Bitbucket.WatchedRepos[1] != "z" {
		t.Fatalf("expected sorted watched repos, got %+v", saved.Bitbucket.WatchedRepos)
	}
}

func TestModifyWatched(t *testing.T) {
	origLoad := loadConfig
	origSave := saveConfig
	defer func() {
		loadConfig = origLoad
		saveConfig = origSave
	}()

	var saved *config.Config
	loadConfig = func() (*config.Config, error) {
		return &config.Config{Bitbucket: config.BitbucketConfig{WatchedRepos: []string{"alpha"}}}, nil
	}
	saveConfig = func(cfg *config.Config) error {
		saved = cfg
		return nil
	}

	out := captureStdout(func() {
		modifyWatched([]string{"beta"}, nil)
	})
	if saved == nil || len(saved.Bitbucket.WatchedRepos) != 2 {
		t.Fatalf("expected watched repos to be updated, got %+v", saved)
	}
	if !strings.Contains(out, "Watched (2): alpha, beta") && !strings.Contains(out, "Watched (2): beta, alpha") {
		t.Fatalf("unexpected modifyWatched output: %q", out)
	}
}

func TestDisplayReposPageWithWatchedRepo(t *testing.T) {
	origLoad := loadConfig
	defer func() { loadConfig = origLoad }()

	loadConfig = func() (*config.Config, error) {
		return &config.Config{Bitbucket: config.BitbucketConfig{WatchedRepos: []string{"repo-a"}}}, nil
	}

	out := captureStdout(func() {
		displayReposPage([]bitbucket.Repository{{Name: "repo-a", FullName: "ws/repo-a", IsPrivate: false, Language: "Go"}}, "workspace")
	})
	if !strings.Contains(out, "[⭐]") || !strings.Contains(out, "repo-a") {
		t.Fatalf("unexpected displayReposPage output: %q", out)
	}
}

func TestDisplayIssueDetailsWithURL(t *testing.T) {
	origLoad := loadConfig
	defer func() { loadConfig = origLoad }()

	loadConfig = func() (*config.Config, error) {
		return &config.Config{Jira: config.JiraConfig{URL: "https://jira.example"}}, nil
	}

	issue := &jira.IssueDetails{Key: "ABC-1"}
	issue.Fields.Summary = "Issue"
	issue.Fields.Status.Name = "Open"
	issue.Fields.Priority.Name = "High"

	out := captureStdout(func() {
		displayIssueDetails(issue)
	})
	if !strings.Contains(out, "https://jira.example/browse/ABC-1") {
		t.Fatalf("expected Jira URL in output, got %q", out)
	}
}

type fakeWorkspacePRLister struct {
	prs []bitbucket.PullRequestWithReviewers
}

func (f fakeWorkspacePRLister) GetWorkspacePullRequestsForUser(username string) ([]bitbucket.PullRequestWithReviewers, error) {
	return f.prs, nil
}

type pagedRepositoryPager struct {
	pages      map[int][]bitbucket.Repository
	totalCount int
	pageCount  int
	lastPage   int
	lastSize   int
}

func (p *pagedRepositoryPager) GetRepositoriesPaged(page, size int) ([]bitbucket.Repository, int, error) {
	p.pageCount++
	p.lastPage = page
	p.lastSize = size
	return p.pages[page], p.totalCount, nil
}

func TestGetPRsToReviewFromAllRepos(t *testing.T) {
	origLoad := loadConfig
	defer func() { loadConfig = origLoad }()

	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Bitbucket: config.BitbucketConfig{
				Workspace:     "workspace",
				BitbucketUser: "bb-user",
			},
		}, nil
	}

	prs, err := getPRsToReviewFromAllRepos(fakeWorkspacePRLister{prs: []bitbucket.PullRequestWithReviewers{{
		ID:    1,
		Title: "Example",
		Source: struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
			Repository struct {
				Name string `json:"name"`
			} `json:"repository"`
		}{Repository: struct {
			Name string `json:"name"`
		}{Name: "repo-a"}},
		Destination: struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
			Repository struct {
				Name string `json:"name"`
			} `json:"repository"`
		}{Repository: struct {
			Name string `json:"name"`
		}{Name: "repo-a"}},
	}}}, "alice", true)
	if err != nil {
		t.Fatalf("getPRsToReviewFromAllRepos failed: %v", err)
	}
	if len(prs) != 1 || prs[0].RepoSlug != "repo-a" {
		t.Fatalf("unexpected PRs: %+v", prs)
	}
}

func TestRunInteractiveMode_Quits(t *testing.T) {
	origLoad := loadConfig
	origMakeRaw := makeRaw
	origRestoreRaw := restoreRaw
	origSave := saveConfig
	origPageSize := pageSize
	origStdin := os.Stdin
	defer func() {
		loadConfig = origLoad
		makeRaw = origMakeRaw
		restoreRaw = origRestoreRaw
		saveConfig = origSave
		pageSize = origPageSize
		os.Stdin = origStdin
	}()

	loadConfig = func() (*config.Config, error) {
		return &config.Config{Bitbucket: config.BitbucketConfig{WatchedRepos: []string{"repo-a"}}}, nil
	}
	makeRaw = func(fd int) (*term.State, error) { return &term.State{}, nil }
	restoreRaw = func(fd int, state *term.State) error { return nil }
	var saved *config.Config
	saveConfig = func(cfg *config.Config) error {
		saved = cfg
		return nil
	}
	pageSize = 10

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	_, _ = w.Write([]byte{27, '[', 'B', ' ', 27, '[', 'C', 27, '[', 'D', 'w', 's', 'x', 'q'})
	_ = w.Close()
	os.Stdin = r

	pager := &pagedRepositoryPager{
		pages: map[int][]bitbucket.Repository{
			0: []bitbucket.Repository{
				{Name: "repo-a", FullName: "ws/repo-a"},
				{Name: "repo-b", FullName: "ws/repo-b"},
			},
			1: []bitbucket.Repository{
				{Name: "repo-c", FullName: "ws/repo-c"},
				{Name: "repo-d", FullName: "ws/repo-d"},
			},
		},
		totalCount: 11,
	}
	captureStdout(func() {
		runInteractiveMode(pager, "workspace")
	})
	if pager.pageCount < 3 {
		t.Fatalf("expected paging interactions, got %d fetches", pager.pageCount)
	}
	if saved == nil || len(saved.Bitbucket.WatchedRepos) != 2 {
		t.Fatalf("expected saved watched repos, got %+v", saved)
	}
}
