package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"devflow/internal/httpx"
	"devflow/internal/jira"
)

func TestListPRsCmdTextAndJSON(t *testing.T) {
	withBitbucketConfig(t)
	registerBitbucketAPI(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2.0/repositories/workspace/repo/pullrequests":
			_ = json.NewEncoder(w).Encode(bitbucket.PullRequestsResponse{
				Values: []bitbucket.PullRequest{
					{
						ID:    1,
						Title: "First PR",
						State: "OPEN",
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	origRepoSlug := repoSlug
	origJSONOutput := jsonOutput
	t.Cleanup(func() {
		repoSlug = origRepoSlug
		jsonOutput = origJSONOutput
	})

	repoSlug = "repo"
	jsonOutput = false
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Bitbucket: config.BitbucketConfig{
				Workspace:    "workspace",
				Token:        "token",
				WatchedRepos: []string{"repo"},
			},
		}, nil
	}

	out := captureStdout(func() {
		listPRsCmd.Run(listPRsCmd, nil)
	})
	if !strings.Contains(out, "Repository: repo (1 PRs)") || !strings.Contains(out, "#1 - First PR") {
		t.Fatalf("unexpected list PR text output: %q", out)
	}

	jsonOutput = true
	out = captureStdout(func() {
		listPRsCmd.Run(listPRsCmd, nil)
	})
	if !strings.Contains(out, "\"workspace\": \"workspace\"") || !strings.Contains(out, "\"total\": 1") {
		t.Fatalf("unexpected list PR json output: %q", out)
	}
}

func TestMyPRsCmdAllReposTextAndJSON(t *testing.T) {
	withBitbucketConfig(t)
	registerBitbucketAPI(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2.0/workspaces/workspace/pullrequests/alice":
			_ = json.NewEncoder(w).Encode(bitbucket.PullRequestsWithReviewersResponse{
				Values: []bitbucket.PullRequestWithReviewers{
					{
						ID:    7,
						Title: "Review me",
						State: "OPEN",
						Source: struct {
							Branch struct {
								Name string `json:"name"`
							} `json:"branch"`
							Repository struct {
								Name string `json:"name"`
							} `json:"repository"`
						}{
							Branch: struct {
								Name string `json:"name"`
							}{Name: "feature"},
							Repository: struct {
								Name string `json:"name"`
							}{Name: "repo-a"},
						},
						Destination: struct {
							Branch struct {
								Name string `json:"name"`
							} `json:"branch"`
							Repository struct {
								Name string `json:"name"`
							} `json:"repository"`
						}{
							Branch: struct {
								Name string `json:"name"`
							}{Name: "main"},
							Repository: struct {
								Name string `json:"name"`
							}{Name: "repo-a"},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	origAllRepos := myPRsAllRepos
	origRepoSlug := myPRsRepoSlug
	origJSON, _ := myPRsCmd.Flags().GetBool("json")
	t.Cleanup(func() {
		myPRsAllRepos = origAllRepos
		myPRsRepoSlug = origRepoSlug
		_ = myPRsCmd.Flags().Set("json", "false")
	})

	myPRsAllRepos = true
	myPRsRepoSlug = ""
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Bitbucket: config.BitbucketConfig{
				Workspace:     "workspace",
				Username:      "alice",
				Token:         "token",
				BitbucketUser: "alice",
			},
		}, nil
	}
	_ = myPRsCmd.Flags().Set("json", "false")

	out := captureStdout(func() {
		myPRsCmd.Run(myPRsCmd, nil)
	})
	if !strings.Contains(out, "all repositories") || !strings.Contains(out, "Review me") {
		t.Fatalf("unexpected my-prs text output: %q", out)
	}

	_ = myPRsCmd.Flags().Set("json", "true")
	out = captureStdout(func() {
		myPRsCmd.Run(myPRsCmd, nil)
	})
	if !strings.Contains(out, "\"total_prs\": 1") || !strings.Contains(out, "\"RepoSlug\": \"repo-a\"") {
		t.Fatalf("unexpected my-prs json output: %q", out)
	}
	_ = myPRsCmd.Flags().Set("json", map[bool]string{true: "true", false: "false"}[origJSON])
}

func TestMyPRsCmdRepoSlugTextAndJSON(t *testing.T) {
	withBitbucketConfig(t)
	registerBitbucketAPI(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2.0/repositories/workspace/repo/pullrequests":
			_ = json.NewEncoder(w).Encode(bitbucket.PullRequestsResponse{
				Values: []bitbucket.PullRequest{
					{ID: 1, Title: "Repo PR", State: "OPEN"},
				},
			})
		case "/2.0/repositories/workspace/repo/pullrequests/1":
			_ = json.NewEncoder(w).Encode(bitbucket.PullRequestDetails{
				ID:    1,
				Title: "Repo PR",
				State: "OPEN",
				Source: struct {
					Branch struct {
						Name string `json:"name"`
					} `json:"branch"`
					Repository struct {
						Name string `json:"name"`
					} `json:"repository"`
				}{
					Branch: struct {
						Name string `json:"name"`
					}{Name: "feature"},
					Repository: struct {
						Name string `json:"name"`
					}{Name: "repo"},
				},
				Destination: struct {
					Branch struct {
						Name string `json:"name"`
					} `json:"branch"`
					Repository struct {
						Name string `json:"name"`
					} `json:"repository"`
				}{
					Branch: struct {
						Name string `json:"name"`
					}{Name: "main"},
					Repository: struct {
						Name string `json:"name"`
					}{Name: "repo"},
				},
				Reviewers: []struct {
					DisplayName string `json:"display_name"`
				}{{DisplayName: "Alice"}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	origRepoSlug := myPRsRepoSlug
	origAllRepos := myPRsAllRepos
	origJSON, _ := myPRsCmd.Flags().GetBool("json")
	t.Cleanup(func() {
		myPRsRepoSlug = origRepoSlug
		myPRsAllRepos = origAllRepos
		_ = myPRsCmd.Flags().Set("json", "false")
	})

	myPRsRepoSlug = "repo"
	myPRsAllRepos = false
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Bitbucket: config.BitbucketConfig{
				Workspace: "workspace",
				Username:  "alice",
				Token:     "token",
				WatchedRepos: []string{"repo"},
			},
		}, nil
	}

	_ = myPRsCmd.Flags().Set("json", "false")
	out := captureStdout(func() {
		myPRsCmd.Run(myPRsCmd, nil)
	})
	if !strings.Contains(out, "Repo PR") || !strings.Contains(out, "repo") {
		t.Fatalf("unexpected my-prs repo text output: %q", out)
	}

	_ = myPRsCmd.Flags().Set("json", "true")
	out = captureStdout(func() {
		myPRsCmd.Run(myPRsCmd, nil)
	})
	if !strings.Contains(out, "\"total_prs\": 1") || !strings.Contains(out, "\"RepoSlug\": \"repo\"") {
		t.Fatalf("unexpected my-prs repo json output: %q", out)
	}
	_ = myPRsCmd.Flags().Set("json", map[bool]string{true: "true", false: "false"}[origJSON])
}

func TestPrCommentsCmdTextAndJSON(t *testing.T) {
	withBitbucketConfig(t)
	registerBitbucketAPI(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2.0/repositories/workspace/repo/pullrequests/1/comments":
			parentID := 1
			_ = json.NewEncoder(w).Encode(bitbucket.CommentsResponse{
				Values: []bitbucket.Comment{
					{
						ID: 1,
						User: struct {
							DisplayName string `json:"display_name"`
							UUID        string `json:"uuid"`
						}{DisplayName: "Alice"},
						CreatedOn: "2026-07-01T10:00:00Z",
						Resolved:  true,
						Content: struct {
							Raw string `json:"raw"`
						}{Raw: "Root comment"},
					},
					{
						ID: 2,
						User: struct {
							DisplayName string `json:"display_name"`
							UUID        string `json:"uuid"`
						}{DisplayName: "Bob"},
						CreatedOn: "2026-07-01T11:00:00Z",
						Parent:    &struct{ ID int `json:"id"` }{ID: parentID},
						Content: struct {
							Raw string `json:"raw"`
						}{Raw: "Reply"},
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Bitbucket: config.BitbucketConfig{
				Workspace: "workspace",
				Username:  "alice",
				Token:     "token",
			},
		}, nil
	}

	origJSON, _ := prCommentsCmd.Flags().GetBool("json")
	t.Cleanup(func() {
		_ = prCommentsCmd.Flags().Set("json", "false")
	})

	_ = prCommentsCmd.Flags().Set("json", "false")
	out := captureStdout(func() {
		prCommentsCmd.Run(prCommentsCmd, []string{"repo", "1"})
	})
	if !strings.Contains(out, "Comments on PR #1") || !strings.Contains(out, "Reply") {
		t.Fatalf("unexpected pr comments text output: %q", out)
	}

	_ = prCommentsCmd.Flags().Set("json", "true")
	out = captureStdout(func() {
		prCommentsCmd.Run(prCommentsCmd, []string{"repo", "1"})
	})
	if !strings.Contains(out, "\"total_comments\": 2") || !strings.Contains(out, "\"total_threads\": 1") {
		t.Fatalf("unexpected pr comments json output: %q", out)
	}
	_ = prCommentsCmd.Flags().Set("json", map[bool]string{true: "true", false: "false"}[origJSON])
}

func TestConfigGetAndSetCommands(t *testing.T) {
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

	out := captureStdout(func() {
		setConfigCmd.Run(setConfigCmd, []string{"bitbucket.workspace", "workspace"})
	})
	if saved == nil || saved.Bitbucket.Workspace != "workspace" {
		t.Fatalf("expected saved config to be updated, got %+v", saved)
	}
	if !strings.Contains(out, "Setting config bitbucket.workspace = workspace") {
		t.Fatalf("unexpected set config output: %q", out)
	}

	cfg := &config.Config{
		Bitbucket: config.BitbucketConfig{Workspace: "workspace"},
	}
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	out = captureStdout(func() {
		getConfigCmd.Run(getConfigCmd, []string{"bitbucket.workspace"})
	})
	if !strings.Contains(out, "bitbucket.workspace = workspace") {
		t.Fatalf("unexpected get config output: %q", out)
	}
}

func TestWatchCommands(t *testing.T) {
	origLoad := loadConfig
	origSave := saveConfig
	defer func() {
		loadConfig = origLoad
		saveConfig = origSave
	}()

	cfg := &config.Config{
		Bitbucket: config.BitbucketConfig{
			WatchedRepos: []string{"beta", "alpha"},
		},
	}
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	saveConfig = func(next *config.Config) error {
		cfg = next
		return nil
	}

	out := captureStdout(func() {
		watchListCmd.Run(watchListCmd, nil)
	})
	if !strings.Contains(out, "Watched repositories:") || !strings.Contains(out, " - alpha") || !strings.Contains(out, " - beta") {
		t.Fatalf("unexpected watch list output: %q", out)
	}

	out = captureStdout(func() {
		watchAddCmd.Run(watchAddCmd, []string{"gamma"})
	})
	if !strings.Contains(out, "Watched (3):") || !strings.Contains(out, "gamma") {
		t.Fatalf("unexpected watch add output: %q", out)
	}

	out = captureStdout(func() {
		watchRemoveCmd.Run(watchRemoveCmd, []string{"beta"})
	})
	if !strings.Contains(out, "Watched (2):") || strings.Contains(out, "beta") {
		t.Fatalf("unexpected watch remove output: %q", out)
	}

	out = captureStdout(func() {
		watchToggleCmd.Run(watchToggleCmd, []string{"alpha", "delta"})
	})
	if !strings.Contains(out, "Watched (2):") || !strings.Contains(out, "delta") {
		t.Fatalf("unexpected watch toggle output: %q", out)
	}
}

func TestSetStatusCmdTextAndJSON(t *testing.T) {
	withBitbucketConfig(t)
	registerBitbucketAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/2.0/repositories/workspace/repo/commit/abcdef1234567890/statuses/build" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bitbucket.CommitStatus{
			State:     "SUCCESSFUL",
			Key:       "ci/build",
			Name:      "CI Build",
			URL:       "https://ci.example/build/1",
			Description: "All tests passed",
			UpdatedOn: "2026-01-01T00:00:00Z",
		})
	})

	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Bitbucket: config.BitbucketConfig{
				Workspace: "workspace",
				Token:     "token",
			},
		}, nil
	}

	origJSON, _ := setStatusCmd.Flags().GetBool("json")
	t.Cleanup(func() {
		_ = setStatusCmd.Flags().Set("json", "false")
	})

	_ = setStatusCmd.Flags().Set("state", "SUCCESSFUL")
	_ = setStatusCmd.Flags().Set("key", "ci/build")
	_ = setStatusCmd.Flags().Set("name", "CI Build")
	_ = setStatusCmd.Flags().Set("url", "https://ci.example/build/1")
	_ = setStatusCmd.Flags().Set("description", "All tests passed")
	_ = setStatusCmd.Flags().Set("json", "false")

	out := captureStdout(func() {
		setStatusCmd.Run(setStatusCmd, []string{"repo", "abcdef1234567890"})
	})
	if !strings.Contains(out, "SUCCESSFUL") || !strings.Contains(out, "CI Build") || !strings.Contains(out, "All tests passed") {
		t.Fatalf("unexpected set-status text output: %q", out)
	}

	_ = setStatusCmd.Flags().Set("json", "true")
	out = captureStdout(func() {
		setStatusCmd.Run(setStatusCmd, []string{"repo", "abcdef1234567890"})
	})
	if !strings.Contains(out, "\"commit_hash\": \"abcdef1234567890\"") || !strings.Contains(out, "\"state\": \"SUCCESSFUL\"") {
		t.Fatalf("unexpected set-status json output: %q", out)
	}
	_ = setStatusCmd.Flags().Set("json", map[bool]string{true: "true", false: "false"}[origJSON])
}

func TestJenkinsBuildsCmd(t *testing.T) {
	host := "jenkins.example"
	httpx.RegisterTestServer(host, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/job/my-job/api/json") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"builds": []map[string]any{
				{
					"number":    42,
					"result":    "SUCCESS",
					"timestamp": int64(1640000000000),
					"duration":  int64(45000),
					"building":  false,
				},
			},
		})
	}))
	t.Cleanup(func() {
		httpx.UnregisterTestServer(host)
	})

	origLoad := loadConfig
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Jenkins: config.JenkinsConfig{URL: "https://" + host},
		}, nil
	}
	t.Cleanup(func() {
		loadConfig = origLoad
	})

	out := captureStdout(func() {
		jenkinsBuildsCmd.Run(jenkinsBuildsCmd, []string{"my-job"})
	})
	if !strings.Contains(out, "Recent builds for job: my-job") || !strings.Contains(out, "✅ SUCCESS") {
		t.Fatalf("unexpected jenkins builds output: %q", out)
	}
}

func TestShowIssueCmd(t *testing.T) {
	host := "jira.example"
	httpx.RegisterTestServer(host, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/ABC-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("fields"); !strings.Contains(got, "summary") || !strings.Contains(got, "comment") {
			t.Fatalf("unexpected fields query: %s", got)
		}
		issue := jira.IssueDetails{Key: "ABC-1"}
		issue.Fields.Summary = "Example issue"
		issue.Fields.Status.Name = "Open"
		issue.Fields.Priority.Name = "High"
		issue.Fields.Assignee.DisplayName = "Alice"
		issue.Fields.Reporter.DisplayName = "Bob"
		issue.Fields.Created = "2026-07-01T10:00:00Z"
		issue.Fields.Updated = "2026-07-02T10:00:00Z"
		issue.Fields.TeamAssigned.Name = "Platform"
		issue.Fields.TeamAssigned.ID = "team-1"
		issue.Fields.Description = map[string]any{"type": "doc", "content": []any{map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "Hello"}}}}}
		issue.Fields.Comment.Comments = []jira.Comment{
			{
				Author: struct {
					DisplayName string `json:"displayName"`
				}{DisplayName: "Carol"},
				Body:    map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "Comment body"}}},
				Created: "2026-07-02T11:00:00Z",
			},
		}
		issue.Fields.Attachment = []jira.Attachment{{Filename: "log.txt", Size: 1024}}
		_ = json.NewEncoder(w).Encode(issue)
	}))
	t.Cleanup(func() {
		httpx.UnregisterTestServer(host)
	})

	origLoad := loadConfig
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Jira: config.JiraConfig{
				URL:      "https://" + host,
				Username: "alice",
				Token:    "token",
			},
		}, nil
	}
	t.Cleanup(func() {
		loadConfig = origLoad
	})

	out := captureStdout(func() {
		showIssueCmd.Run(showIssueCmd, []string{"ABC-1"})
	})
	if !strings.Contains(out, "ABC-1") || !strings.Contains(out, "Example issue") || !strings.Contains(out, "Attachments (1)") {
		t.Fatalf("unexpected jira show output: %q", out)
	}
}
