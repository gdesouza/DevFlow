package cmd

import (
	"fmt"
	"log"
	"strings"

	"devflow/internal/config"
	"devflow/internal/jira"
	"github.com/spf13/cobra"
)

var showIssueCmd = &cobra.Command{
	Use:   "show [issue-key]",
	Short: "Show detailed information about a Jira issue",
	Long:  `Display comprehensive details about a specific Jira issue including description, status, priority, assignee, and more`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey := args[0]

		// Load configuration
		cfg, err := config.Load()
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

		// Display issue details
		displayIssueDetails(issue)
	},
}

func init() {
	// This will be called when the jira command is initialized
}

func displayIssueDetails(issue *jira.IssueDetails) {
	// Load config to get the base URL
	cfg, _ := config.Load()
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
