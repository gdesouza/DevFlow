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
			if showPullRequests {
				pullRequests, err = client.GetIssuePullRequests(issue.ID)
				if err != nil {
					log.Fatalf("Error fetching pull requests: %v", err)
				}
			}
			if err := printJSON(issueJSONValue(issue, pullRequests, showPullRequests)); err != nil {
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

func init() {
	showIssueCmd.Flags().BoolVar(&showChildren, "children", false, "List the ticket's child items")
	showIssueCmd.Flags().BoolVar(&showPullRequests, "pull-requests", false, "List the ticket's linked pull requests")
}

func issueJSONValue(issue *jira.IssueDetails, pullRequests []jira.PullRequestRef, includePullRequests bool) any {
	if !includePullRequests {
		return issue
	}
	if pullRequests == nil {
		pullRequests = []jira.PullRequestRef{}
	}
	return struct {
		*jira.IssueDetails
		PullRequests []jira.PullRequestRef `json:"pull_requests"`
	}{
		IssueDetails: issue,
		PullRequests: pullRequests,
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
	jql := fmt.Sprintf("parent = %s ORDER BY status ASC, priority DESC", issueKey)
	children, err := client.Search(jql, true, 0, 0)
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
