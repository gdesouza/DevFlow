package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"devflow/internal/config"
	"devflow/internal/jira"
	"github.com/spf13/cobra"
)

var (
	createProjectKey      string
	createIssueType       string
	createPriority        string
	createAssignee        string
	createLabels          string
	createEpic            string
	createStoryPoints     float64
	createSprint          string
	createDescription     string
	createDescriptionFile string
)

var createTaskCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Create a new Jira task",
	Long:  `Create a new Jira task with the specified title and optional metadata`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		title := args[0]

		if createProjectKey == "" {
			log.Fatal("--project is required (Jira project key)")
		}

		// Load config
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
		if cfg.Jira.URL == "" || cfg.Jira.Username == "" || cfg.Jira.Token == "" {
			log.Fatal("Jira configuration incomplete. Run: devflow config set jira.url|jira.username|jira.token ...")
		}

		description, err := resolveDescription(createDescription, createDescriptionFile)
		if err != nil {
			log.Fatalf("Failed to read description: %v", err)
		}

		labels := parseLabels(createLabels)

		client := jira.NewClient(&cfg.Jira)

		issue, err := client.CreateIssue(jira.CreateIssueOptions{
			ProjectKey:  createProjectKey,
			Summary:     title,
			Description: description,
			IssueType:   createIssueType,
			Priority:    createPriority,
			Assignee:    createAssignee,
			Labels:      labels,
			Epic:        createEpic,
			StoryPoints: createStoryPoints,
			Sprint:      createSprint,
		})
		if err != nil {
			log.Fatalf("Failed to create Jira issue: %v", err)
		}

		fmt.Printf("Created Jira issue %s: %s\n", issue.Key, title)
		fmt.Printf("URL: %s/browse/%s\n", cfg.Jira.URL, issue.Key)
	},
}

func init() {
	createTaskCmd.Flags().StringVarP(&createProjectKey, "project", "p", "", "Jira project key (required)")
	createTaskCmd.Flags().StringVarP(&createIssueType, "type", "t", "Task", "Issue type (Task, Story, Bug, etc.)")
	createTaskCmd.Flags().StringVar(&createPriority, "priority", "", "Priority (Highest, High, Medium, Low, Lowest)")
	createTaskCmd.Flags().StringVar(&createAssignee, "assignee", "", "Assignee username (may require accountId in some instances)")
	createTaskCmd.Flags().StringVar(&createLabels, "labels", "", "Comma-separated labels (e.g. backend,api,urgent)")
	createTaskCmd.Flags().StringVar(&createEpic, "epic", "", "Epic key to link (depends on Jira setup)")
	createTaskCmd.Flags().Float64Var(&createStoryPoints, "story-points", 0, "Story points estimate (will use common custom field id)")
	createTaskCmd.Flags().StringVar(&createSprint, "sprint", "", "Sprint name or ID (depends on Jira setup)")
	createTaskCmd.Flags().StringVarP(&createDescription, "description", "d", "", "Issue description text")
	createTaskCmd.Flags().StringVar(&createDescriptionFile, "description-file", "", "Path to file for description body")
}

func parseLabels(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func resolveDescription(inline, filePath string) (string, error) {
	if inline != "" && filePath != "" {
		return "", errors.New("specify either --description or --description-file, not both")
	}
	if filePath != "" {
		data, err := os.ReadFile(filepath.Clean(filePath))
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return inline, nil
}
