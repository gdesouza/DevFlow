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
	pullRequests[0].Source.Repository.Name = "mapping-service"

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

func TestNormalizedPullRequestIncludesRepository(t *testing.T) {
	issue := &jira.IssueDetails{ID: "359571", Key: "PSS-3369"}
	pullRequests := []jira.PullRequestRef{{ID: "pr-1"}}
	pullRequests[0].Source.Repository.Name = "mapping-service"

	encoded, err := json.Marshal(normalizedIssueJSON(issue, pullRequests, true, nil, false))
	if err != nil {
		t.Fatalf("marshal normalized issue JSON: %v", err)
	}

	var output struct {
		PullRequests []struct {
			Repository string `json:"repository"`
		} `json:"pull_requests"`
	}
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("unmarshal normalized issue JSON: %v", err)
	}
	if len(output.PullRequests) != 1 || output.PullRequests[0].Repository != "mapping-service" {
		t.Fatalf("repository was not included: %+v", output.PullRequests)
	}
}

func TestNormalizedIssueJSONWithTreeIncludesChildPullRequests(t *testing.T) {
	issue := &jira.IssueDetails{ID: "1", Key: "EPIC-1"}
	child := jira.Issue{ID: "2", Key: "TASK-1"}
	child.Fields.Summary = "Child task"
	child.Fields.Status.Name = "In Progress"
	pullRequests := []jira.PullRequestRef{{ID: "pr-2", Name: "Child PR"}}

	encoded, err := json.Marshal(normalizedIssueJSONWithTree(issue, nil, true, []issueTreeNode{{
		Issue:        child,
		PullRequests: pullRequests,
	}}))
	if err != nil {
		t.Fatalf("marshal recursive issue JSON: %v", err)
	}

	var output struct {
		Children []struct {
			Key          string `json:"key"`
			PullRequests []struct {
				ID string `json:"id"`
			} `json:"pull_requests"`
		} `json:"children"`
	}
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("unmarshal recursive issue JSON: %v", err)
	}
	if len(output.Children) != 1 || output.Children[0].Key != "TASK-1" || len(output.Children[0].PullRequests) != 1 || output.Children[0].PullRequests[0].ID != "pr-2" {
		t.Fatalf("recursive child pull requests were not included: %+v", output.Children)
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

func TestIssueJSONValueIncludesChildrenWhenRequested(t *testing.T) {
	issue := &jira.IssueDetails{ID: "359571", Key: "PSS-3369"}
	children := []jira.Issue{{Key: "PSS-3370"}}

	encoded, err := json.Marshal(issueJSONValueWithChildren(issue, nil, false, children, true))
	if err != nil {
		t.Fatalf("marshal issue JSON: %v", err)
	}

	var output struct {
		Children []jira.Issue `json:"children"`
	}
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("unmarshal issue JSON: %v", err)
	}
	if len(output.Children) != 1 || output.Children[0].Key != "PSS-3370" {
		t.Fatalf("children were not included: %+v", output.Children)
	}
}

func TestNormalizedIssueJSONFlattensFieldsAndADF(t *testing.T) {
	issue := &jira.IssueDetails{ID: "359571", Key: "PSS-3369"}
	issue.Fields.Summary = "Remote Control"
	issue.Fields.Status.Name = "In Progress"
	issue.Fields.Priority.Name = "Medium"
	issue.Fields.Assignee.DisplayName = "Matthew Moore"
	issue.Fields.Description = map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type":    "paragraph",
				"content": []any{map[string]any{"type": "text", "text": "Hello"}},
			},
		},
	}

	encoded, err := json.Marshal(normalizedIssueJSON(issue, nil, false, nil, false))
	if err != nil {
		t.Fatalf("marshal normalized issue JSON: %v", err)
	}

	var output map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("unmarshal normalized issue JSON: %v", err)
	}
	if _, ok := output["fields"]; ok {
		t.Fatal("normalized JSON should not contain a fields wrapper")
	}
	var description string
	if err := json.Unmarshal(output["description"], &description); err != nil {
		t.Fatalf("decode normalized description: %v", err)
	}
	if description != "Hello" {
		t.Fatalf("normalized description = %q, want %q", description, "Hello")
	}
}
