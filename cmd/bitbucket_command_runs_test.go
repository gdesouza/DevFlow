package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"devflow/internal/httpx"
)

func registerBitbucketHost(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	httpx.RegisterTestServer("api.bitbucket.org", handler)
	t.Cleanup(func() { httpx.UnregisterTestServer("api.bitbucket.org") })
}

func setBitbucketCmdConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	orig := loadConfig
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	t.Cleanup(func() { loadConfig = orig })
}

func TestParticipatingCmdSingleRepo(t *testing.T) {
	setBitbucketCmdConfig(t, &config.Config{
		Bitbucket: config.BitbucketConfig{
			Workspace:     "workspace",
			Username:      "alice",
			Token:         "token",
			WatchedRepos:  []string{"repo"},
			BitbucketUser: "alice",
		},
	})
	registerBitbucketHost(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/repositories/workspace/repo/pullrequests" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("q"); !strings.Contains(got, "participants.username") {
			t.Fatalf("unexpected query: %s", got)
		}
		_ = json.NewEncoder(w).Encode(bitbucket.PullRequestsResponse{
			Values: []bitbucket.PullRequest{{ID: 1, Title: "PR", State: "OPEN"}},
		})
	})

	origRepo := participatingRepoSlug
	origJSON, _ := participatingCmd.Flags().GetBool("json")
	defer func() {
		participatingRepoSlug = origRepo
		_ = participatingCmd.Flags().Set("json", "false")
	}()
	participatingRepoSlug = "repo"
	_ = participatingCmd.Flags().Set("json", "false")

	out := captureStdout(func() {
		participatingCmd.Run(participatingCmd, nil)
	})
	if !strings.Contains(out, "Found 1 pull requests in 'repo'") || !strings.Contains(out, "#1 - PR") {
		t.Fatalf("unexpected participating output: %q", out)
	}

	_ = participatingCmd.Flags().Set("json", "true")
	out = captureStdout(func() {
		participatingCmd.Run(participatingCmd, nil)
	})
	if !strings.Contains(out, "\"total_prs\": 1") || !strings.Contains(out, "\"repository\": \"repo\"") {
		t.Fatalf("unexpected participating json output: %q", out)
	}
	_ = participatingCmd.Flags().Set("json", map[bool]string{true: "true", false: "false"}[origJSON])
}

func TestSearchReposCmd(t *testing.T) {
	setBitbucketCmdConfig(t, &config.Config{
		Bitbucket: config.BitbucketConfig{Workspace: "workspace", Token: "token"},
	})
	registerBitbucketHost(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/repositories/workspace" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bitbucket.RepositoriesResponse{
			Values: []bitbucket.Repository{
				{Name: "api-service", Description: "gRPC API"},
				{Name: "infra", Description: "terraform"},
			},
		})
	})

	origCase := caseSensitive
	origDesc := includeDescription
	defer func() {
		caseSensitive = origCase
		includeDescription = origDesc
	}()
	caseSensitive = false
	includeDescription = true

	out := captureStdout(func() {
		searchReposCmd.Run(searchReposCmd, []string{"api|terraform"})
	})
	if !strings.Contains(out, "Found 2 matching repositories") || !strings.Contains(out, "api-service") || !strings.Contains(out, "infra") {
		t.Fatalf("unexpected search output: %q", out)
	}
}

func TestReadmeListAndRepoShowCmds(t *testing.T) {
	setBitbucketCmdConfig(t, &config.Config{
		Bitbucket: config.BitbucketConfig{Workspace: "workspace", Username: "alice", Token: "token"},
	})
	registerBitbucketHost(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/2.0/repositories/workspace" && r.URL.Query().Get("page") == "1":
			_ = json.NewEncoder(w).Encode(bitbucket.RepositoriesResponse{
				Values: []bitbucket.Repository{{Name: "repo", FullName: "workspace/repo"}},
				Size:   1,
			})
		case r.URL.Path == "/2.0/repositories/workspace/repo":
			_ = json.NewEncoder(w).Encode(bitbucket.Repository{
				Name:        "repo",
				FullName:    "workspace/repo",
				Description: "desc",
				IsPrivate:   true,
				Language:    "Go",
				Size:        1024,
			})
		case r.URL.Path == "/2.0/repositories/workspace/repo/src/HEAD/README.md":
			_, _ = w.Write([]byte("# Hello"))
		default:
			t.Fatalf("unexpected path: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	})

	out := captureStdout(func() {
		readmeCmd.Run(readmeCmd, []string{"repo"})
	})
	if !strings.Contains(out, "README (README.md)") || !strings.Contains(out, "# Hello") {
		t.Fatalf("unexpected readme output: %q", out)
	}

	origInteractive := interactive
	origStartPage := startPage
	origPageSize := pageSize
	defer func() {
		interactive = origInteractive
		startPage = origStartPage
		pageSize = origPageSize
	}()
	interactive = false
	startPage = 1
	pageSize = 20
	out = captureStdout(func() {
		listReposCmd.Run(listReposCmd, nil)
	})
	if !strings.Contains(out, "Found 1 repositories in workspace 'workspace'") || !strings.Contains(out, "repo") {
		t.Fatalf("unexpected list repos output: %q", out)
	}

	out = captureStdout(func() {
		showRepoCmd.Run(showRepoCmd, []string{"repo"})
	})
	if !strings.Contains(out, "Repository: repo") || !strings.Contains(out, "Language: Go") {
		t.Fatalf("unexpected repo show output: %q", out)
	}
}

func TestCreateShowDiffAndCommentCommands(t *testing.T) {
	setBitbucketCmdConfig(t, &config.Config{
		Bitbucket: config.BitbucketConfig{Workspace: "workspace", Username: "alice", Token: "token"},
	})
	registerBitbucketHost(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/workspace/repo":
			_ = json.NewEncoder(w).Encode(bitbucket.Repository{MainBranch: struct {
				Name string `json:"name"`
			}{Name: "main"}})
		case r.Method == http.MethodPost && r.URL.Path == "/2.0/repositories/workspace/repo/pullrequests":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(bitbucket.PullRequest{
				ID:    9,
				Title: "New PR",
				Description: "body",
				Author: struct {
					DisplayName string `json:"display_name"`
				}{DisplayName: "Alice"},
				Source: struct {
					Branch struct {
						Name string `json:"name"`
					} `json:"branch"`
				}{Branch: struct {
					Name string `json:"name"`
				}{Name: "feature"}},
				Destination: struct {
					Branch struct {
						Name string `json:"name"`
					} `json:"branch"`
				}{Branch: struct {
					Name string `json:"name"`
				}{Name: "main"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/workspace/repo/pullrequests/9":
			_ = json.NewEncoder(w).Encode(bitbucket.PullRequestDetails{
				ID:    9,
				Title: "New PR",
				State: "OPEN",
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
			})
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/workspace/repo/pullrequests/9/diff":
			_, _ = w.Write([]byte("diff --git a/a b/a"))
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/workspace/repo/pullrequests/9/comments":
			_ = json.NewEncoder(w).Encode(bitbucket.CommentsResponse{Values: []bitbucket.Comment{{ID: 1, Content: struct {
				Raw string `json:"raw"`
			}{Raw: "comment"}}}})
		case r.Method == http.MethodPost && r.URL.Path == "/2.0/repositories/workspace/repo/pullrequests/9/comments":
			_ = json.NewEncoder(w).Encode(bitbucket.Comment{ID: 2})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	origRepo := prRepoSlug
	origSource := sourceBranch
	origDest := destinationBranch
	origDesc := prDescription
	origReviewers := append([]string{}, prReviewers...)
	defer func() {
		prRepoSlug = origRepo
		sourceBranch = origSource
		destinationBranch = origDest
		prDescription = origDesc
		prReviewers = origReviewers
	}()

	prRepoSlug = "repo"
	sourceBranch = "feature"
	destinationBranch = "main"
	prDescription = "body"
	prReviewers = []string{"bob"}

	out := captureStdout(func() {
		createPRCmd.Run(createPRCmd, []string{"New PR"})
	})
	if !strings.Contains(out, "Successfully created pull request") || !strings.Contains(out, "#9 - New PR") {
		t.Fatalf("unexpected create pr output: %q", out)
	}

	out = captureStdout(func() {
		showPRCmd.Run(showPRCmd, []string{"repo", "9"})
	})
	if !strings.Contains(out, "New PR") || !strings.Contains(out, "Branches: feature → main") {
		t.Fatalf("unexpected show pr output: %q", out)
	}

	out = captureStdout(func() {
		prDiffCmd.Run(prDiffCmd, []string{"repo", "9"})
	})
	if !strings.Contains(out, "diff --git") {
		t.Fatalf("unexpected diff output: %q", out)
	}

	out = captureStdout(func() {
		addCommentCmd.Run(addCommentCmd, []string{"repo", "9", "hello"})
	})
	if !strings.Contains(out, "Comment added successfully") {
		t.Fatalf("unexpected add-comment output: %q", out)
	}

	out = captureStdout(func() {
		commentReplyCmd.Run(commentReplyCmd, []string{"repo", "9", "1", "reply"})
	})
	if !strings.Contains(out, "Successfully replied to thread") {
		t.Fatalf("unexpected comment-reply output: %q", out)
	}
}
