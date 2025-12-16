package cmd

import (
	"fmt"
	"log"

	"devflow/internal/config"
	"devflow/internal/jira"
	"github.com/spf13/cobra"
)

var (
	// Command flags
	filterStatus string
	sortBy       string
	showPriority bool
	showSprint   bool
	excludeDone  bool
	// Search flags
	searchQuery string
	searchJQL   string
	maxResults  int
	page        int
)

var listTasksCmd = &cobra.Command{
	Use:   "list",
	Short: "List Jira tasks",
	Long:  `List all Jira tasks assigned to the current user`,
	Run: func(cmd *cobra.Command, args []string) {
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

		// Get issues
		var issues []jira.Issue

		// Compute startAt from page (1-based). If page <= 0, startAtArg will be 0.
		startAtArg := 0
		if page > 1 && maxResults > 0 {
			startAtArg = (page - 1) * maxResults
		}

		if searchJQL != "" {
			// Raw JQL provided
			iss, err := client.Search(searchJQL, true, maxResults, startAtArg)
			if err != nil {
				log.Fatalf("Error searching Jira issues with JQL: %v", err)
			}
			issues = iss
		} else if searchQuery != "" {
			// Free text search
			iss, err := client.Search(searchQuery, false, maxResults, startAtArg)
			if err != nil {
				log.Fatalf("Error searching Jira issues: %v", err)
			}
			issues = iss
		} else {
			// Default: issues assigned to current user
			iss, err := client.GetMyIssues()
			if err != nil {
				log.Fatalf("Error fetching Jira issues: %v", err)
			}
			issues = iss
		}

		// Apply filters
		filteredIssues := filterIssues(issues, filterStatus, excludeDone)

		// Sort issues
		sortedIssues := sortIssues(filteredIssues, sortBy)

		// Display results
		if len(sortedIssues) == 0 {
			fmt.Println("No Jira tasks found matching your criteria.")
			return
		}

		fmt.Printf("Found %d Jira tasks assigned to you", len(sortedIssues))
		if filterStatus != "" {
			fmt.Printf(" (filtered by status: %s)", filterStatus)
		}
		if excludeDone {
			fmt.Printf(" (excluding done tasks)")
		}
		fmt.Printf(":\n\n")

		for _, issue := range sortedIssues {
			statusIcon := getStatusIcon(issue.Fields.Status.Name)
			fmt.Printf("%s %s - %s", statusIcon, issue.Key, issue.Fields.Summary)

			if showPriority && issue.Fields.Priority.Name != "" {
				priorityIcon := getPriorityIcon(issue.Fields.Priority.Name)
				fmt.Printf(" %s", priorityIcon)
			}

			fmt.Printf(" 🔗 %s/browse/%s\n", cfg.Jira.URL, issue.Key)

			if showSprint {
				if sprintName := getSprintName(issue.Fields.Sprint); sprintName != "" {
					fmt.Printf("   📅 Sprint: %s\n", sprintName)
				}
			}
		}
	},
}

func init() {
	listTasksCmd.Flags().StringVarP(&filterStatus, "filter", "f", "", "Filter by status (e.g., 'In Progress', 'Done')")
	listTasksCmd.Flags().StringVarP(&sortBy, "sort", "s", "status", "Sort by: status, priority, updated")
	listTasksCmd.Flags().BoolVarP(&showPriority, "priority", "p", false, "Show task priority")
	listTasksCmd.Flags().BoolVarP(&showSprint, "sprint", "r", false, "Show sprint information")
	listTasksCmd.Flags().BoolVar(&excludeDone, "exclude-done", false, "Exclude completed/done tasks")
	// Search flags
	listTasksCmd.Flags().StringVarP(&searchQuery, "query", "q", "", "Free-text search query (will be converted to JQL)")
	listTasksCmd.Flags().StringVar(&searchJQL, "jql", "", "Raw JQL query (takes precedence over --query)")
	listTasksCmd.Flags().IntVar(&maxResults, "max-results", 0, "Maximum number of results to return (0 = use server default)")
	listTasksCmd.Flags().IntVar(&page, "page", 0, "Page number to retrieve (1-based). Use with --max-results")
}

// filterIssues filters issues based on status and exclude done flag
func filterIssues(issues []jira.Issue, filterStatus string, excludeDone bool) []jira.Issue {
	var filtered []jira.Issue

	for _, issue := range issues {
		// Filter by status if specified
		if filterStatus != "" && issue.Fields.Status.Name != filterStatus {
			continue
		}

		// Exclude done tasks if flag is set
		if excludeDone && isDoneStatus(issue.Fields.Status.Name) {
			continue
		}

		filtered = append(filtered, issue)
	}

	return filtered
}

// sortIssues sorts issues based on the specified criteria
func sortIssues(issues []jira.Issue, sortBy string) []jira.Issue {
	// Simple sorting - in a real implementation you might want more sophisticated sorting
	switch sortBy {
	case "status":
		// Sort by status priority (active tasks first)
		return sortByStatusPriority(issues)
	case "priority":
		// Sort by priority (highest first)
		return sortByPriority(issues)
	case "updated":
		// Already sorted by updated date from API
		return issues
	default:
		return issues
	}
}

// isDoneStatus checks if a status indicates the task is completed
func isDoneStatus(status string) bool {
	doneStatuses := []string{"Done", "Closed", "Resolved", "Cancelled"}
	for _, doneStatus := range doneStatuses {
		if status == doneStatus {
			return true
		}
	}
	return false
}

// sortByStatusPriority sorts issues with active statuses first
func sortByStatusPriority(issues []jira.Issue) []jira.Issue {
	// Define status priority order (higher number = higher priority)
	statusPriority := map[string]int{
		"In Progress":         10,
		"In Review":           9,
		"To Do":               8,
		"Open":                7,
		"Backlog":             6,
		"Scoping":             5,
		"Under investigation": 4,
		"Blocked":             3,
		"Waiting for support": 2,
		"Done":                1,
		"Closed":              1,
		"Resolved":            1,
		"Cancelled":           1,
	}

	// Simple bubble sort by priority
	sorted := make([]jira.Issue, len(issues))
	copy(sorted, issues)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			priority1 := statusPriority[sorted[j].Fields.Status.Name]
			priority2 := statusPriority[sorted[j+1].Fields.Status.Name]

			if priority1 < priority2 {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// sortByPriority sorts issues by priority level
func sortByPriority(issues []jira.Issue) []jira.Issue {
	// Define priority order (higher number = higher priority)
	priorityOrder := map[string]int{
		"Highest": 5,
		"High":    4,
		"Medium":  3,
		"Low":     2,
		"Lowest":  1,
	}

	sorted := make([]jira.Issue, len(issues))
	copy(sorted, issues)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			priority1 := priorityOrder[sorted[j].Fields.Priority.Name]
			priority2 := priorityOrder[sorted[j+1].Fields.Priority.Name]

			if priority1 < priority2 {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// getPriorityIcon returns an icon for the given priority
func getPriorityIcon(priority string) string {
	switch priority {
	case "Highest":
		return "🔴"
	case "High":
		return "🟠"
	case "Medium":
		return "🟡"
	case "Low":
		return "🟢"
	case "Lowest":
		return "🔵"
	default:
		return "⚪"
	}
}

// getSprintName extracts sprint name from the sprint field
func getSprintName(sprint interface{}) string {
	// Sprint field can be an array or object, this is a simplified implementation
	if sprint == nil {
		return ""
	}

	// For now, return a placeholder - in a real implementation you'd parse the sprint data
	return "Current Sprint"
}

// getStatusIcon returns an appropriate emoji/icon for the given Jira status
func getStatusIcon(status string) string {
	switch status {
	case "To Do", "Open":
		return "📋"
	case "In Progress", "In Review":
		return "🔄"
	case "Done", "Closed", "Resolved":
		return "✅"
	case "Backlog":
		return "📚"
	case "Blocked", "Waiting for support":
		return "🚫"
	case "Under investigation":
		return "🔍"
	case "Scoping":
		return "📏"
	case "Cancelled":
		return "❌"
	default:
		return "📝"
	}
}
