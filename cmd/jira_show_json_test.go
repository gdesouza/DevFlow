package cmd

import (
	"encoding/json"
	"testing"

	"devflow/internal/jira"
)

func TestIssueJSONValueIncludesPullRequestsWhenRequested(t *testing.T) {
	issue := &jira.IssueDetails{ID: "359571", Key: "PSS-3369"}
	pullRequests := []jira.PullRequestRef{{
		ID:     "pr-1",
		Name:   "Add remote control",
		URL:    "https://bitbucket.org/example/repo/pull-requests/1",
		Status: "OPEN",
	}}

	encoded, err := json.Marshal(issueJSONValue(issue, pullRequests, true))
	if err != nil {
		t.Fatalf("marshal issue JSON: %v", err)
	}

	var output struct {
		ID           string                `json:"id"`
		Key          string                `json:"key"`
		PullRequests []jira.PullRequestRef `json:"pull_requests"`
	}
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("unmarshal issue JSON: %v", err)
	}
	if output.ID != issue.ID || output.Key != issue.Key {
		t.Fatalf("issue identity was not preserved: %+v", output)
	}
	if len(output.PullRequests) != 1 || output.PullRequests[0].ID != "pr-1" {
		t.Fatalf("pull requests were not included: %+v", output.PullRequests)
	}
}

func TestIssueJSONValuePreservesDefaultShapeWithoutPullRequests(t *testing.T) {
	issue := &jira.IssueDetails{ID: "359571", Key: "PSS-3369"}

	encoded, err := json.Marshal(issueJSONValue(issue, nil, false))
	if err != nil {
		t.Fatalf("marshal issue JSON: %v", err)
	}

	var output map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("unmarshal issue JSON: %v", err)
	}
	if _, ok := output["pull_requests"]; ok {
		t.Fatal("pull_requests should be omitted unless requested")
	}
}
