package cmd

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"devflow/internal/config"
	"devflow/internal/jira"
	"github.com/spf13/cobra"
)

var (
	updateAssignee        string
	updatePriority        string
	updateLabels          string
	updateSummary         string
	updateTitle           string
	updateDescription     string
	updateDescriptionFile string
	updateEpic            string
	updateStoryPoints     float64
	updateSprint          string
	updateTeam            string
)

var updateTaskCmd = &cobra.Command{
	Use:   "update [issue-key]",
	Short: "Update fields on a Jira issue",
	Long:  "Update one or more fields on an existing Jira issue (assignee, priority, labels, summary, description, epic, story-points, sprint, team).",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey := args[0]

		// At least one flag must be provided
		if updateAssignee == "" && updatePriority == "" && updateLabels == "" && updateSummary == "" &&
			updateDescription == "" && updateDescriptionFile == "" && updateEpic == "" &&
			updateStoryPoints == 0 && updateSprint == "" && updateTeam == "" && updateTitle == "" {
			log.Fatal("Provide at least one field to update (use --help for flags)")
		}

		// Title alias
		if updateTitle != "" && updateSummary == "" {
			updateSummary = updateTitle
		}

		// Resolve description (mutually exclusive)
		description, err := resolveDescription(updateDescription, updateDescriptionFile)
		if err != nil {
			log.Fatalf("Failed to read description: %v", err)
		}

		// Load config
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
		if cfg.Jira.URL == "" || cfg.Jira.Username == "" || cfg.Jira.Token == "" {
			log.Fatal("Jira configuration incomplete. Run: devflow config set jira.url|jira.username|jira.token ...")
		}

		client := jira.NewClient(&cfg.Jira)

		fields := make(map[string]interface{})

		if updateSummary != "" {
			fields["summary"] = updateSummary
		}
		if description != "" {
			fields["description"] = description
		}
		if updatePriority != "" {
			fields["priority"] = map[string]string{"name": updatePriority}
		}
		if updateAssignee != "" {
			fields["assignee"] = map[string]string{"name": updateAssignee}
		}
		if strings.TrimSpace(updateLabels) != "" {
			fields["labels"] = parseLabels(updateLabels)
		}
		if updateStoryPoints > 0 {
			fields["customfield_10016"] = updateStoryPoints
		}
		if updateEpic != "" {
			fields["customfield_10014"] = updateEpic
		}
		if updateSprint != "" {
			fields["customfield_10020"] = updateSprint
		}
		if updateTeam != "" {
			// mimic CreateIssue behavior: accept numeric id or string name
			if _, err := strconv.Atoi(updateTeam); err == nil {
				fields["customfield_11887"] = map[string]string{"id": updateTeam}
			} else {
				fields["customfield_11887"] = map[string]string{"name": updateTeam}
			}
		}

		if err := client.UpdateIssue(issueKey, fields); err != nil {
			log.Fatalf("Failed to update issue: %v", err)
		}
		fmt.Printf("✅ Updated %s\n", issueKey)
	},
}

func init() {
	updateTaskCmd.Flags().StringVar(&updateAssignee, "assignee", "", "Assignee username (may require accountId in some instances)")
	updateTaskCmd.Flags().StringVar(&updatePriority, "priority", "", "Priority (Highest, High, Medium, Low, Lowest)")
	updateTaskCmd.Flags().StringVar(&updateLabels, "labels", "", "Comma-separated labels (e.g. backend,api,urgent)")
	updateTaskCmd.Flags().StringVar(&updateSummary, "summary", "", "New summary/title")
	updateTaskCmd.Flags().StringVar(&updateTitle, "title", "", "Alias for --summary (for ergonomics)")
	updateTaskCmd.Flags().StringVar(&updateDescription, "description", "", "Issue description text")
	updateTaskCmd.Flags().StringVar(&updateDescriptionFile, "description-file", "", "Path to file for description body")
	updateTaskCmd.Flags().StringVar(&updateEpic, "epic", "", "Epic key to link")
	updateTaskCmd.Flags().Float64Var(&updateStoryPoints, "story-points", 0, "Story points value")
	updateTaskCmd.Flags().StringVar(&updateSprint, "sprint", "", "Sprint name or ID")
	updateTaskCmd.Flags().StringVar(&updateTeam, "team", "", "Team id or name for Team Assigned custom field")
}
