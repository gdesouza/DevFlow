package bitbucket

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"devflow/internal/config"
)

func TestCommentMethods(t *testing.T) {
	var receivedPaths []string
	var nextURL string
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPaths = append(receivedPaths, r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, "/comments") && r.Method == http.MethodGet:
			if r.URL.RawQuery == "" {
				resp := CommentsResponse{
					Values: []Comment{{
						ID: 1,
						Content: struct {
							Raw string `json:"raw"`
						}{Raw: "hello"},
						User: struct {
							DisplayName string `json:"display_name"`
							UUID        string `json:"uuid"`
						}{DisplayName: "Alice", UUID: "u1"},
					}},
					Next: nextURL,
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			_ = json.NewEncoder(w).Encode(CommentsResponse{Values: []Comment{}})
		case strings.HasSuffix(r.URL.Path, "/comments") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(Comment{ID: 2})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	nextURL = server.URL + "/2.0/repositories/ws/repo/pullrequests/1/comments?page=2"

	client := NewClient(&config.BitbucketConfig{Workspace: "ws", Token: "tok"})
	client.baseURL = server.URL + "/2.0"

	comments, err := client.GetPullRequestComments("repo", 1)
	if err != nil {
		t.Fatalf("GetPullRequestComments failed: %v", err)
	}
	if len(comments) != 1 || comments[0].ID != 1 {
		t.Fatalf("unexpected comments: %+v", comments)
	}

	comment, err := client.CreatePullRequestComment("repo", 1, "hello")
	if err != nil || comment.ID != 2 {
		t.Fatalf("CreatePullRequestComment failed: %+v err=%v", comment, err)
	}

	inline, err := client.CreatePullRequestInlineComment("repo", 1, "inline", "file.go", 42)
	if err != nil || inline.ID != 2 {
		t.Fatalf("CreatePullRequestInlineComment failed: %+v err=%v", inline, err)
	}

	reply, err := client.ReplyToPullRequestComment("repo", 1, 99, "reply")
	if err != nil || reply.ID != 2 {
		t.Fatalf("ReplyToPullRequestComment failed: %+v err=%v", reply, err)
	}

	if len(receivedPaths) < 4 {
		t.Fatalf("expected all comment methods to hit the server, got %v", receivedPaths)
	}
}

func TestPipelineMethods(t *testing.T) {
	var nextURL string
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/pipelines/") && r.Method == http.MethodGet:
			if r.URL.RawQuery == "" || strings.Contains(r.URL.RawQuery, "pagelen=2") {
				resp := PipelinesResponse{
					Values: []Pipeline{{
						UUID:        "{pipeline-1}",
						BuildNumber: 7,
						State: struct {
							Name  string `json:"name"`
							Stage *struct {
								Name string `json:"name"`
							} `json:"stage"`
							Result *struct {
								Name string `json:"name"`
							} `json:"result"`
						}{Name: "COMPLETED", Result: &struct {
							Name string `json:"name"`
						}{Name: "SUCCESSFUL"}},
						Target: struct {
							RefName string `json:"ref_name"`
							RefType string `json:"ref_type"`
							Commit  *struct {
								Hash string `json:"hash"`
							} `json:"commit"`
						}{RefName: "main", Commit: &struct {
							Hash string `json:"hash"`
						}{Hash: "abcdef"}},
					}},
					Next: nextURL,
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			_ = json.NewEncoder(w).Encode(PipelinesResponse{Values: []Pipeline{}})
		case strings.Contains(r.URL.Path, "/steps/") && strings.HasSuffix(r.URL.Path, "/log"):
			_, _ = io.WriteString(w, "pipeline log")
		case strings.Contains(r.URL.Path, "/steps/"):
			_ = json.NewEncoder(w).Encode(PipelineStepsResponse{
				Values: []PipelineStep{{
					UUID:              "{step-1}",
					Name:              "build",
					DurationInSeconds: 125,
					State: struct {
						Name   string `json:"name"`
						Result *struct {
							Name string `json:"name"`
						} `json:"result"`
					}{Name: "COMPLETED", Result: &struct {
						Name string `json:"name"`
					}{Name: "FAILED"}},
				}},
			})
		case strings.Contains(r.URL.Path, "/pipelines/") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(Pipeline{
				UUID:        "{pipeline-1}",
				BuildNumber: 7,
				State: struct {
					Name  string `json:"name"`
					Stage *struct {
						Name string `json:"name"`
					} `json:"stage"`
					Result *struct {
						Name string `json:"name"`
					} `json:"result"`
				}{Name: "COMPLETED", Result: &struct {
					Name string `json:"name"`
				}{Name: "SUCCESSFUL"}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	nextURL = server.URL + "/2.0/repositories/ws/repo/pipelines/?page=2"

	client := NewClient(&config.BitbucketConfig{Workspace: "ws", Token: "tok"})
	client.baseURL = server.URL + "/2.0"

	pipelines, err := client.GetPipelines("repo", 2)
	if err != nil || len(pipelines) != 1 {
		t.Fatalf("GetPipelines failed: %+v err=%v", pipelines, err)
	}

	pipeline, err := client.GetPipeline("repo", "{pipeline-1}")
	if err != nil || pipeline.UUID != "{pipeline-1}" {
		t.Fatalf("GetPipeline failed: %+v err=%v", pipeline, err)
	}

	steps, err := client.GetPipelineSteps("repo", "{pipeline-1}")
	if err != nil || len(steps) != 1 {
		t.Fatalf("GetPipelineSteps failed: %+v err=%v", steps, err)
	}

	logOut, err := client.GetPipelineStepLog("repo", "{pipeline-1}", "{step-1}")
	if err != nil || logOut != "pipeline log" {
		t.Fatalf("GetPipelineStepLog failed: %q err=%v", logOut, err)
	}
}
