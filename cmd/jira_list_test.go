package cmd

import (
	"devflow/internal/jira"
	"testing"
)

func makeIssueWithStatus(status string) jira.Issue {
	var fields jira.Issue
	fields.Fields.Status.Name = status
	return jira.Issue{
		Fields: fields.Fields,
	}
}

func makeIssueWithPriority(priority string) jira.Issue {
	var fields jira.Issue
	fields.Fields.Priority.Name = priority
	return jira.Issue{
		Fields: fields.Fields,
	}
}

func TestFilterIssues(t *testing.T) {
	tests := []struct {
		name         string
		issues       []jira.Issue
		filterStatus string
		excludeDone  bool
		wantCount    int
	}{
		{
			name: "no filter, no exclude",
			issues: []jira.Issue{
				makeIssueWithStatus("To Do"),
				makeIssueWithStatus("Done"),
			},
			filterStatus: "",
			excludeDone:  false,
			wantCount:    2,
		},
		{
			name: "filter by status",
			issues: []jira.Issue{
				makeIssueWithStatus("In Progress"),
				makeIssueWithStatus("Done"),
			},
			filterStatus: "In Progress",
			excludeDone:  false,
			wantCount:    1,
		},
		{
			name: "exclude done",
			issues: []jira.Issue{
				makeIssueWithStatus("In Progress"),
				makeIssueWithStatus("Done"),
			},
			filterStatus: "",
			excludeDone:  true,
			wantCount:    1,
		},
		{
			name: "filter and exclude done",
			issues: []jira.Issue{
				makeIssueWithStatus("Done"),
				makeIssueWithStatus("To Do"),
			},
			filterStatus: "To Do",
			excludeDone:  true,
			wantCount:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterIssues(tt.issues, tt.filterStatus, tt.excludeDone)
			if len(got) != tt.wantCount {
				t.Errorf("filterIssues() got %d issues, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestIsDoneStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"Done", true},
		{"Closed", true},
		{"Resolved", true},
		{"Cancelled", true},
		{"In Progress", false},
		{"Open", false},
		{"To Do", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := isDoneStatus(tt.status)
			if got != tt.want {
				t.Errorf("isDoneStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestGetPriorityIcon(t *testing.T) {
	tests := []struct {
		priority string
		want     string
	}{
		{"Highest", "🔴"},
		{"High", "🟠"},
		{"Medium", "🟡"},
		{"Low", "🟢"},
		{"Lowest", "🔵"},
		{"Unknown", "⚪"},
		{"", "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			got := getPriorityIcon(tt.priority)
			if got != tt.want {
				t.Errorf("getPriorityIcon(%q) = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}

func TestSortIssues_StatusPriority(t *testing.T) {
	issues := []jira.Issue{
		makeIssueWithStatus("Done"),
		makeIssueWithStatus("In Progress"),
		makeIssueWithStatus("To Do"),
		makeIssueWithStatus("Closed"),
		makeIssueWithStatus("In Review"),
		makeIssueWithStatus("Scoping"),
	}
	sorted := sortIssues(issues, "status")
	// The highest status priorities are "In Progress" (10), "In Review" (9), etc.
	statusOrder := []string{"In Progress", "In Review", "To Do", "Scoping", "Done", "Closed"}
	for i, status := range statusOrder {
		if sorted[i].Fields.Status.Name != status {
			t.Errorf("sortIssues by status: got %v at %d, want %v", sorted[i].Fields.Status.Name, i, status)
		}
	}
}

func TestSortIssues_Priority(t *testing.T) {
	issues := []jira.Issue{
		makeIssueWithPriority("Lowest"),
		makeIssueWithPriority("Medium"),
		makeIssueWithPriority("High"),
		makeIssueWithPriority("Highest"),
		makeIssueWithPriority("Low"),
	}
	sorted := sortIssues(issues, "priority")
	order := []string{"Highest", "High", "Medium", "Low", "Lowest"}
	for i, priority := range order {
		if sorted[i].Fields.Priority.Name != priority {
			t.Errorf("sortIssues by priority: got %v at %d, want %v", sorted[i].Fields.Priority.Name, i, priority)
		}
	}
}

func TestGetSprintName(t *testing.T) {
	tests := []struct {
		input interface{}
		want  string
	}{
		{nil, ""},
		{"ignored", "Current Sprint"},
		{[]interface{}{map[string]interface{}{"name": "Sprint 1"}}, "Current Sprint"},
	}
	for _, tt := range tests {
		t.Run("sprint", func(t *testing.T) {
			got := getSprintName(tt.input)
			if got != tt.want {
				t.Errorf("getSprintName(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"To Do", "📋"},
		{"Open", "📋"},
		{"In Progress", "🔄"},
		{"In Review", "🔄"},
		{"Done", "✅"},
		{"Closed", "✅"},
		{"Resolved", "✅"},
		{"Backlog", "📚"},
		{"Blocked", "🚫"},
		{"Waiting for support", "🚫"},
		{"Under investigation", "🔍"},
		{"Scoping", "📏"},
		{"Cancelled", "❌"},
		{"OtherStatus", "📝"},
		{"", "📝"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := getStatusIcon(tt.input)
			if got != tt.want {
				t.Errorf("getStatusIcon(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
