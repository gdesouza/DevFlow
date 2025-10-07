package cmd

import (
	"github.com/spf13/cobra"
)

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Work items (Jira)",
	Long:  `Manage Jira tasks, issues, and workflow related actions`,
}

func init() {
	tasksCmd.AddCommand(listTasksCmd)
	tasksCmd.AddCommand(createTaskCmd)
	tasksCmd.AddCommand(showIssueCmd)
	tasksCmd.AddCommand(mentionedCmd)
	tasksCmd.AddCommand(commentCmd)
	tasksCmd.AddCommand(linkCmd)
}
