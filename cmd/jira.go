package cmd

import (
	"github.com/spf13/cobra"
)

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Work items (Jira)",
	Long: `Manage Jira tasks, issues, and workflow related actions.

Available actions:
  list        List your assigned issues
  show        Show detailed issue information (includes Assigned Team)
  mentioned   Find issues where you are mentioned
  create      Create a new issue (supports epic, story points, sprint, team, labels)
  comment     Add a comment to an issue
  link        Add an external document / remote link to an issue`,
}

func init() {
	tasksCmd.AddCommand(listTasksCmd)
	tasksCmd.AddCommand(createTaskCmd)
	tasksCmd.AddCommand(showIssueCmd)
	tasksCmd.AddCommand(mentionedCmd)
	tasksCmd.AddCommand(commentCmd)
	tasksCmd.AddCommand(linkCmd)
}
