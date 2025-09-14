package cmd

import (
	"github.com/spf13/cobra"
)

var jiraCmd = &cobra.Command{
	Use:   "jira",
	Short: "Jira operations",
	Long:  `Manage Jira tasks, issues, and workflows`,
}

func init() {
	jiraCmd.AddCommand(listTasksCmd)
	jiraCmd.AddCommand(createTaskCmd)
	jiraCmd.AddCommand(showIssueCmd)
	jiraCmd.AddCommand(mentionedCmd)
}
