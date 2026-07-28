package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"devflow/internal/jira"
)

func TestNormalizedText(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "plain text", value: "  hello\nworld  ", want: "hello\nworld"},
		{
			name: "ADF text and link",
			value: map[string]any{
				"type": "doc",
				"content": []any{map[string]any{
					"content": []any{
						map[string]any{"text": "See"},
						map[string]any{"attrs": map[string]any{"url": "https://example.com"}},
					},
				}},
			},
			want: "See https://example.com",
		},
		{name: "nil", value: nil, want: ""},
		{name: "fallback", value: 42, want: "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizedText(tt.value); got != tt.want {
				t.Fatalf("normalizedText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizedChildFromTreeIncludesNestedChildrenAndPullRequests(t *testing.T) {
	grandchild := jira.Issue{Key: "TASK-2"}
	grandchild.Fields.Summary = "Grandchild"
	grandchild.Fields.Status.Name = "Done"
	child := jira.Issue{Key: "TASK-1"}
	child.Fields.Summary = "Child"
	child.Fields.Status.Name = "In Progress"

	got := normalizedChildFromTree(issueTreeNode{
		Issue:        child,
		PullRequests: []jira.PullRequestRef{{ID: "pr-1", Name: "Child PR"}},
		Children:     []issueTreeNode{{Issue: grandchild}},
	}, true)

	if got.Key != "TASK-1" || got.Status != "In Progress" {
		t.Fatalf("unexpected child: %+v", got)
	}
	if got.PullRequests == nil || len(*got.PullRequests) != 1 || (*got.PullRequests)[0].ID != "pr-1" {
		t.Fatalf("child pull requests missing: %+v", got.PullRequests)
	}
	if got.Children == nil || len(*got.Children) != 1 || (*got.Children)[0].Key != "TASK-2" {
		t.Fatalf("nested children missing: %+v", got.Children)
	}
}

func TestRawIssueTreeValueIncludesRequestedEmptyCollections(t *testing.T) {
	issue := &jira.IssueDetails{ID: "1", Key: "EPIC-1"}
	value := rawIssueTreeValue(issue, nil, true, []issueTreeNode{{Issue: jira.Issue{Key: "TASK-1"}}})
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal raw tree: %v", err)
	}

	var output map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("unmarshal raw tree: %v", err)
	}
	for _, field := range []string{"pull_requests", "children"} {
		if _, ok := output[field]; !ok {
			t.Fatalf("raw tree missing %q: %s", field, encoded)
		}
	}
	if got := string(output["pull_requests"]); got != "[]" {
		t.Fatalf("empty pull requests = %s, want []", got)
	}
}

func TestDisplayRecursiveTable(t *testing.T) {
	issue := &jira.IssueDetails{Key: "EPIC-1"}
	issue.Fields.Summary = "Epic"
	issue.Fields.Status.Name = "Open"
	child := jira.Issue{Key: "TASK-1"}
	child.Fields.Summary = "Child"
	child.Fields.Status.Name = "Done"

	out := captureStdout(func() {
		displayRecursiveTable(issue, []jira.PullRequestRef{{ID: "pr-1"}}, true, []issueTreeNode{{Issue: child}})
	})
	for _, want := range []string{"TICKET", "NAME", "STATUS", "PRS", "EPIC-1", "TASK-1", "Done"} {
		if !strings.Contains(out, want) {
			t.Fatalf("recursive table missing %q: %s", want, out)
		}
	}
}

func TestTabularCellEscapesLineBreaks(t *testing.T) {
	if got := tabularCell("first\r\nsecond"); got != "first\\nsecond" {
		t.Fatalf("tabularCell() = %q, want %q", got, "first\\nsecond")
	}
}
