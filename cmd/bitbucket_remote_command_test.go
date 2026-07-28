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

func withBitbucketConfig(t *testing.T) {
	t.Helper()

	origLoad := loadConfig
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Bitbucket: config.BitbucketConfig{
				Workspace: "workspace",
				Token:     "token",
			},
		}, nil
	}
	t.Cleanup(func() {
		loadConfig = origLoad
	})
}

func registerBitbucketAPI(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	httpx.RegisterTestServer("api.bitbucket.org", handler)
	t.Cleanup(func() {
		httpx.UnregisterTestServer("api.bitbucket.org")
	})
}

func TestPipelinesListCommand(t *testing.T) {
	withBitbucketConfig(t)
	registerBitbucketAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/repositories/workspace/repo/pipelines/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bitbucket.PipelinesResponse{
			Values: []bitbucket.Pipeline{
				{
					UUID:             "{pipeline-1}",
					BuildNumber:      42,
					CreatedOn:        "2026-01-01T10:00:00Z",
					BuildSecondsUsed: 125,
				},
			},
		})
	})

	t.Cleanup(func() {
		_ = pipelinesListCmd.Flags().Set("limit", "10")
		_ = pipelinesListCmd.Flags().Set("json", "false")
	})
	_ = pipelinesListCmd.Flags().Set("limit", "1")
	_ = pipelinesListCmd.Flags().Set("json", "false")

	out := captureStdout(func() {
		pipelinesListCmd.Run(pipelinesListCmd, []string{"repo"})
	})

	if !strings.Contains(out, "Fetching pipelines for workspace/repo") || !strings.Contains(out, "42") || !strings.Contains(out, "2m5s") {
		t.Fatalf("unexpected pipelines list output: %q", out)
	}

	_ = pipelinesListCmd.Flags().Set("json", "true")
	out = captureStdout(func() {
		pipelinesListCmd.Run(pipelinesListCmd, []string{"repo"})
	})
	if !strings.Contains(out, "\"build_number\": 42") || !strings.Contains(out, "\"uuid\": \"{pipeline-1}\"") {
		t.Fatalf("unexpected pipelines list json output: %q", out)
	}
}

func TestPipelinesShowAndLogCommands(t *testing.T) {
	withBitbucketConfig(t)
	registerBitbucketAPI(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/2.0/repositories/workspace/repo/pipelines/" && strings.Contains(r.URL.RawQuery, "pagelen=100"):
			_ = json.NewEncoder(w).Encode(bitbucket.PipelinesResponse{
				Values: []bitbucket.Pipeline{
					{UUID: "{pipeline-1}", BuildNumber: 42},
				},
			})
		case r.URL.Path == "/2.0/repositories/workspace/repo/pipelines/{pipeline-1}":
			_ = json.NewEncoder(w).Encode(bitbucket.Pipeline{
				UUID:             "{pipeline-1}",
				BuildNumber:      42,
				CreatedOn:        "2026-01-01T10:00:00Z",
				CompletedOn:      "2026-01-01T10:02:00Z",
				BuildSecondsUsed: 120,
			})
		case r.URL.Path == "/2.0/repositories/workspace/repo/pipelines/{pipeline-1}/steps/":
			_ = json.NewEncoder(w).Encode(bitbucket.PipelineStepsResponse{
				Values: []bitbucket.PipelineStep{
					{
						UUID:              "{step-1}",
						Name:              "Build",
						DurationInSeconds: 61,
					},
				},
			})
		case r.URL.Path == "/2.0/repositories/workspace/repo/pipelines/{pipeline-1}/steps/{step-1}/log":
			_, _ = w.Write([]byte("step log output"))
		default:
			t.Fatalf("unexpected path: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	})

	t.Cleanup(func() {
		_ = pipelinesShowCmd.Flags().Set("json", "false")
	})
	_ = pipelinesShowCmd.Flags().Set("json", "false")

	out := captureStdout(func() {
		pipelinesShowCmd.Run(pipelinesShowCmd, []string{"repo", "42"})
	})
	if !strings.Contains(out, "Pipeline #42") || !strings.Contains(out, "Build") || !strings.Contains(out, "To view a step's log") {
		t.Fatalf("unexpected pipelines show output: %q", out)
	}

	_ = pipelinesShowCmd.Flags().Set("json", "true")
	out = captureStdout(func() {
		pipelinesShowCmd.Run(pipelinesShowCmd, []string{"repo", "42"})
	})
	if !strings.Contains(out, "\"pipeline\"") || !strings.Contains(out, "\"steps\"") {
		t.Fatalf("unexpected pipelines show json output: %q", out)
	}

	out = captureStdout(func() {
		pipelinesLogCmd.Run(pipelinesLogCmd, []string{"repo", "{pipeline-1}", "{step-1}"})
	})
	if !strings.Contains(out, "step log output") {
		t.Fatalf("unexpected pipelines log output: %q", out)
	}
}

func TestBuildsCommand(t *testing.T) {
	withBitbucketConfig(t)
	registerBitbucketAPI(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2.0/repositories/workspace/repo/pullrequests/42/commits":
			_ = json.NewEncoder(w).Encode(bitbucket.CommitsResponse{
				Values: []bitbucket.Commit{
					{
						Hash:    "abcdef1234567890",
						Message: "Commit title\nMore details",
						Date:    "2026-01-01T00:00:00Z",
						Author: struct {
							Raw string `json:"raw"`
						}{Raw: "Alice"},
					},
				},
			})
		case "/2.0/repositories/workspace/repo/commit/abcdef1234567890/statuses":
			_ = json.NewEncoder(w).Encode(bitbucket.CommitStatusesResponse{
				Values: []bitbucket.CommitStatus{
					{
						State:     "SUCCESSFUL",
						Key:       "ci/build",
						Name:      "CI Build",
						URL:       "https://ci.example/build/1",
						UpdatedOn: "2026-01-01T00:00:00Z",
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	})

	t.Cleanup(func() {
		_ = buildsCmd.Flags().Set("json", "false")
	})
	_ = buildsCmd.Flags().Set("json", "false")

	out := captureStdout(func() {
		buildsCmd.Run(buildsCmd, []string{"repo", "42"})
	})
	if !strings.Contains(out, "Commit 1/1") || !strings.Contains(out, "CI Build") || !strings.Contains(out, "https://ci.example/build/1") {
		t.Fatalf("unexpected builds output: %q", out)
	}
}
