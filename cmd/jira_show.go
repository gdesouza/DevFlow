package cmd

import (
	"fmt"
	"log"
	"strings"

	"devflow/internal/config"
	"devflow/internal/jira"
	"github.com/spf13/cobra"
)

var showChildren bool
var showPullRequests bool

var showIssueCmd = &cobra.Command{
	Use:   "show [issue-key]",
	Short: "Show detailed information about a Jira issue",
	Long:  `Display comprehensive details about a specific Jira issue including description, status, priority, assignee, and more`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey := args[0]

		// Load configuration
		cfg, err := loadConfig()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		// Validate required config
		if cfg.Jira.URL == "" {
			log.Fatal("Jira URL not configured. Run: devflow config set jira.url <url>")
		}
		if cfg.Jira.Username == "" {
			log.Fatal("Jira username not configured. Run: devflow config set jira.username <username>")
		}
		if cfg.Jira.Token == "" {
			log.Fatal("Jira token not configured. Run: devflow config set jira.token <token>")
		}

		// Create Jira client
		client := jira.NewClient(&cfg.Jira)

		// Get issue details
		issue, err := client.GetIssueDetails(issueKey)
		if err != nil {
			log.Fatalf("Error fetching issue details: %v", err)
		}
		if wantsJSON(cmd) {
			var pullRequests []jira.PullRequestRef
			var children []jira.Issue
			if showPullRequests {
				pullRequests, err = client.GetIssuePullRequests(issue.ID)
				if err != nil {
					log.Fatalf("Error fetching pull requests: %v", err)
				}
			}
			if showChildren {
				children, err = fetchChildIssues(client, issueKey)
				if err != nil {
					log.Fatalf("Error fetching child issues: %v", err)
				}
			}
			var output any
			if wantsRaw(cmd) {
				output = issueJSONValueWithChildren(issue, pullRequests, showPullRequests, children, showChildren)
			} else {
				output = normalizedIssueJSON(issue, pullRequests, showPullRequests, children, showChildren)
			}
			if err := printJSON(output); err != nil {
				log.Fatalf("Error encoding JSON: %v", err)
			}
			return
		}

		// Display issue details
		displayIssueDetails(issue)

		if showChildren {
			displayChildIssues(client, cfg, issueKey)
		}

		if showPullRequests {
			displayPullRequests(client, issue)
		}
	},
}

type normalizedIssue struct {
	ID           string                   `json:"id"`
	Key          string                   `json:"key"`
	Summary      string                   `json:"summary"`
	Status       string                   `json:"status"`
	Priority     string                   `json:"priority,omitempty"`
	Assignee     string                   `json:"assignee,omitempty"`
	Reporter     string                   `json:"reporter,omitempty"`
	Team         *normalizedTeam          `json:"team,omitempty"`
	Created      string                   `json:"created,omitempty"`
	Updated      string                   `json:"updated,omitempty"`
	Description  string                   `json:"description,omitempty"`
	Comments     []normalizedComment      `json:"comments"`
	Attachments  []normalizedAttachment   `json:"attachments"`
	Children     *[]normalizedChild       `json:"children,omitempty"`
	PullRequests *[]normalizedPullRequest `json:"pull_requests,omitempty"`
}

type normalizedTeam struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type normalizedComment struct {
	Author  string `json:"author"`
	Created string `json:"created"`
	Body    string `json:"body"`
}

type normalizedAttachment struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Created  string `json:"created,omitempty"`
}

type normalizedChild struct {
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
	Priority string `json:"priority,omitempty"`
	Assignee string `json:"assignee,omitempty"`
}

type normalizedPullRequest struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Repository        string `json:"repository,omitempty"`
	URL               string `json:"url,omitempty"`
	Status            string `json:"status,omitempty"`
	Author            string `json:"author,omitempty"`
	SourceBranch      string `json:"source_branch,omitempty"`
	DestinationBranch string `json:"destination_branch,omitempty"`
	LastUpdate        string `json:"last_update,omitempty"`
}

func normalizedIssueJSON(issue *jira.IssueDetails, pullRequests []jira.PullRequestRef, includePullRequests bool, children []jira.Issue, includeChildren bool) normalizedIssue {
	comments := make([]normalizedComment, 0, len(issue.Fields.Comment.Comments))
	for _, comment := range issue.Fields.Comment.Comments {
		comments = append(comments, normalizedComment{
			Author:  comment.Author.DisplayName,
			Created: comment.Created,
			Body:    normalizedText(comment.Body),
		})
	}

	attachments := make([]normalizedAttachment, 0, len(issue.Fields.Attachment))
	for _, attachment := range issue.Fields.Attachment {
		attachments = append(attachments, normalizedAttachment{
			Filename: attachment.Filename,
			Size:     attachment.Size,
			Created:  attachment.Created,
		})
	}

	output := normalizedIssue{
		ID:          issue.ID,
		Key:         issue.Key,
		Summary:     issue.Fields.Summary,
		Status:      issue.Fields.Status.Name,
		Priority:    issue.Fields.Priority.Name,
		Assignee:    issue.Fields.Assignee.DisplayName,
		Reporter:    issue.Fields.Reporter.DisplayName,
		Created:     issue.Fields.Created,
		Updated:     issue.Fields.Updated,
		Description: normalizedText(issue.Fields.Description),
		Comments:    comments,
		Attachments: attachments,
	}
	if issue.Fields.TeamAssigned.ID != "" || issue.Fields.TeamAssigned.Name != "" {
		output.Team = &normalizedTeam{ID: issue.Fields.TeamAssigned.ID, Name: issue.Fields.TeamAssigned.Name}
	}

	if includeChildren {
		normalizedChildren := make([]normalizedChild, 0, len(children))
		for _, child := range children {
			normalizedChildren = append(normalizedChildren, normalizedChild{
				Key:      child.Key,
				Summary:  child.Fields.Summary,
				Status:   child.Fields.Status.Name,
				Priority: child.Fields.Priority.Name,
				Assignee: child.Fields.Assignee.DisplayName,
			})
		}
		output.Children = &normalizedChildren
	}

	if includePullRequests {
		normalizedPullRequests := make([]normalizedPullRequest, 0, len(pullRequests))
		for _, pullRequest := range pullRequests {
			repository := pullRequest.Source.Repository.Name
			if repository == "" {
				repository = pullRequest.Destination.Repository.Name
			}
			normalizedPullRequests = append(normalizedPullRequests, normalizedPullRequest{
				ID:                pullRequest.ID,
				Name:              pullRequest.Name,
				Repository:        repository,
				URL:               pullRequest.URL,
				Status:            pullRequest.Status,
				Author:            pullRequest.Author.Name,
				SourceBranch:      pullRequest.Source.Branch,
				DestinationBranch: pullRequest.Destination.Branch,
				LastUpdate:        pullRequest.LastUpdate,
			})
		}
		output.PullRequests = &normalizedPullRequests
	}

	return output
}

func normalizedText(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return cleanDescription(text)
	}

	var parts []string
	var walk func(any)
	walk = func(current any) {
		switch item := current.(type) {
		case map[string]any:
			if text, ok := item["text"].(string); ok {
				parts = append(parts, text)
			}
			if attrs, ok := item["attrs"].(map[string]any); ok {
				if url, ok := attrs["url"].(string); ok {
					parts = append(parts, url)
				}
			}
			if content, ok := item["content"]; ok {
				walk(content)
			}
		case []any:
			for _, child := range item {
				walk(child)
			}
		}
	}
	walk(value)
	if len(parts) > 0 {
		return strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
	}
	return cleanDescription(fmt.Sprintf("%v", value))
}

func init() {
	showIssueCmd.Flags().BoolVar(&showChildren, "children", false, "List the ticket's child items")
	showIssueCmd.Flags().BoolVar(&showPullRequests, "pull-requests", false, "List the ticket's linked pull requests")
}

func issueJSONValue(issue *jira.IssueDetails, pullRequests []jira.PullRequestRef, includePullRequests bool) any {
	return issueJSONValueWithChildren(issue, pullRequests, includePullRequests, nil, false)
}

func issueJSONValueWithChildren(issue *jira.IssueDetails, pullRequests []jira.PullRequestRef, includePullRequests bool, children []jira.Issue, includeChildren bool) any {
	var pullRequestsValue *[]jira.PullRequestRef
	if includePullRequests {
		if pullRequests == nil {
			pullRequests = []jira.PullRequestRef{}
		}
		pullRequestsValue = &pullRequests
	}

	var childrenValue *[]jira.Issue
	if includeChildren {
		if children == nil {
			children = []jira.Issue{}
		}
		childrenValue = &children
	}

	if !includePullRequests && !includeChildren {
		return issue
	}
	return struct {
		*jira.IssueDetails
		PullRequests *[]jira.PullRequestRef `json:"pull_requests,omitempty"`
		Children     *[]jira.Issue          `json:"children,omitempty"`
	}{
		IssueDetails: issue,
		PullRequests: pullRequestsValue,
		Children:     childrenValue,
	}
}

// displayPullRequests fetches and prints the pull requests linked to the
// given issue via the Jira dev-status integration.
func displayPullRequests(client *jira.Client, issue *jira.IssueDetails) {
	prs, err := client.GetIssuePullRequests(issue.ID)
	if err != nil {
		log.Fatalf("Error fetching pull requests: %v", err)
	}

	fmt.Println()
	fmt.Printf("🔀 Pull Requests (%d):\n", len(prs))
	fmt.Println(strings.Repeat("─", 80))

	if len(prs) == 0 {
		fmt.Println("No pull requests found.")
		return
	}

	for _, pr := range prs {
		fmt.Printf("%s %s (%s → %s)\n", getPRStatusIcon(pr.Status), pr.Name, pr.Source.Branch, pr.Destination.Branch)
		if pr.Author.Name != "" {
			fmt.Printf("   👤 %s", pr.Author.Name)
		}
		if pr.URL != "" {
			fmt.Printf(" 🔗 %s", pr.URL)
		}
		fmt.Println()
	}
}

// displayChildIssues fetches and prints the child issues (e.g. subtasks or
// items whose parent is issueKey) for the given ticket.
func displayChildIssues(client *jira.Client, cfg *config.Config, issueKey string) {
	children, err := fetchChildIssues(client, issueKey)
	if err != nil {
		log.Fatalf("Error fetching child issues: %v", err)
	}

	fmt.Println()
	fmt.Printf("🧩 Child Items (%d):\n", len(children))
	fmt.Println(strings.Repeat("─", 80))

	if len(children) == 0 {
		fmt.Println("No child items found.")
		return
	}

	for _, child := range children {
		statusIcon := getStatusIcon(child.Fields.Status.Name)
		fmt.Printf("%s %s - %s", statusIcon, child.Key, child.Fields.Summary)
		if child.Fields.Priority.Name != "" {
			fmt.Printf(" %s", getPriorityIcon(child.Fields.Priority.Name))
		}
		if cfg != nil && cfg.Jira.URL != "" {
			fmt.Printf(" 🔗 %s/browse/%s", cfg.Jira.URL, child.Key)
		}
		fmt.Println()
	}
}

func fetchChildIssues(client *jira.Client, issueKey string) ([]jira.Issue, error) {
	jql := fmt.Sprintf("parent = %s ORDER BY status ASC, priority DESC", issueKey)
	return client.Search(jql, true, 0, 0)
}

func displayIssueDetails(issue *jira.IssueDetails) {
	// Load config to get the base URL
	cfg, _ := loadConfig()
	if cfg != nil && cfg.Jira.URL != "" {
		fmt.Printf("🔹 %s: %s 🔗 %s/browse/%s\n", issue.Key, issue.Fields.Summary, cfg.Jira.URL, issue.Key)
	} else {
		fmt.Printf("🔹 %s: %s\n", issue.Key, issue.Fields.Summary)
	}

	fmt.Println(strings.Repeat("=", 80))

	// Status and Priority
	statusIcon := getStatusIcon(issue.Fields.Status.Name)
	priorityIcon := getPriorityIcon(issue.Fields.Priority.Name)
	fmt.Printf("📊 Status: %s %s\n", statusIcon, issue.Fields.Status.Name)
	fmt.Printf("🎯 Priority: %s %s\n", priorityIcon, issue.Fields.Priority.Name)

	// Assignee
	if issue.Fields.Assignee.DisplayName != "" {
		fmt.Printf("👤 Assignee: %s\n", issue.Fields.Assignee.DisplayName)
	} else {
		fmt.Printf("👤 Assignee: Unassigned\n")
	}

	// Reporter
	if issue.Fields.Reporter.DisplayName != "" {
		fmt.Printf("📝 Reporter: %s\n", issue.Fields.Reporter.DisplayName)
	}

	// Assigned Team
	if issue.Fields.TeamAssigned.Name != "" || issue.Fields.TeamAssigned.ID != "" {
		label := issue.Fields.TeamAssigned.Name
		if label == "" {
			label = issue.Fields.TeamAssigned.ID
		}
		fmt.Printf("👥 Assigned Team: %s (id: %s)\n", label, issue.Fields.TeamAssigned.ID)
	}

	// Created and Updated dates
	if issue.Fields.Created != "" {
		fmt.Printf("📅 Created: %s\n", issue.Fields.Created)
	}
	if issue.Fields.Updated != "" {
		fmt.Printf("🔄 Updated: %s\n", issue.Fields.Updated)
	}

	fmt.Println()

	// Description
	if issue.Fields.Description != nil {
		fmt.Println("📄 Description:")
		fmt.Println("─────────────")
		// Convert interface{} to string and clean up
		descStr := fmt.Sprintf("%v", issue.Fields.Description)
		cleanDescription := cleanDescription(descStr)
		fmt.Println(cleanDescription)
		fmt.Println()
	}

	// Comments
	if len(issue.Fields.Comment.Comments) > 0 {
		fmt.Printf("💬 Comments (%d):\n", len(issue.Fields.Comment.Comments))
		fmt.Println("────────────")
		for i, comment := range issue.Fields.Comment.Comments {
			fmt.Printf("%d. %s - %s\n", i+1, comment.Author.DisplayName, comment.Created)
			// Handle comment body as interface{} in case it's complex
			bodyStr := fmt.Sprintf("%v", comment.Body)
			fmt.Printf("   %s\n\n", cleanDescription(bodyStr))
		}
	}

	// Attachments
	if len(issue.Fields.Attachment) > 0 {
		fmt.Printf("📎 Attachments (%d):\n", len(issue.Fields.Attachment))
		fmt.Println("─────────────")
		for _, attachment := range issue.Fields.Attachment {
			fmt.Printf("• %s (%s)\n", attachment.Filename, formatFileSize(attachment.Size))
		}
		fmt.Println()
	}
}

func cleanDescription(description string) string {
	// Handle Atlassian Document Format (ADF) - extract text content
	if strings.Contains(description, "type:") && strings.Contains(description, "content:") {
		return extractTextFromADF(description)
	}

	// Simple HTML tag removal and formatting for regular HTML
	description = strings.ReplaceAll(description, "<br>", "\n")
	description = strings.ReplaceAll(description, "<br/>", "\n")
	description = strings.ReplaceAll(description, "<p>", "")
	description = strings.ReplaceAll(description, "</p>", "\n")
	description = strings.ReplaceAll(description, "<strong>", "")
	description = strings.ReplaceAll(description, "</strong>", "")
	description = strings.ReplaceAll(description, "<em>", "")
	description = strings.ReplaceAll(description, "</em>", "")

	// Remove other common HTML tags
	tags := []string{"<b>", "</b>", "<i>", "</i>", "<u>", "</u>", "<ul>", "</ul>", "<ol>", "</ol>", "<li>", "</li>"}
	for _, tag := range tags {
		description = strings.ReplaceAll(description, tag, "")
	}

	return strings.TrimSpace(description)
}

func extractTextFromADF(adfString string) string {
	var result strings.Builder
	var inText bool

	// Simple ADF text extraction - look for "text:" patterns
	parts := strings.Split(adfString, "text:")
	for i, part := range parts {
		if i == 0 {
			continue // Skip the first part
		}

		// Find the end of this text segment
		endIndex := strings.Index(part, " type:")
		if endIndex == -1 {
			endIndex = strings.Index(part, " content:")
		}
		if endIndex == -1 {
			endIndex = strings.Index(part, " marks:")
		}
		if endIndex == -1 {
			endIndex = len(part)
		}

		text := part[:endIndex]
		// Remove quotes if present
		text = strings.Trim(text, `"`)

		if text != "" {
			if inText {
				result.WriteString(" ")
			}
			result.WriteString(text)
			inText = true
		}
	}

	// If we couldn't extract meaningful text, return a simplified version
	if result.Len() == 0 {
		// Remove some ADF noise and return a basic representation
		simplified := strings.ReplaceAll(adfString, "map[", "")
		simplified = strings.ReplaceAll(simplified, "]", "")
		simplified = strings.ReplaceAll(simplified, " type:", "\n")
		simplified = strings.ReplaceAll(simplified, " content:", "")
		simplified = strings.ReplaceAll(simplified, " text:", " ")
		return simplified
	}

	return result.String()
}

func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
