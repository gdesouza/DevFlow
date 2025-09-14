package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var createTaskCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Create a new Jira task",
	Long:  `Create a new Jira task with the specified title`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		title := args[0]
		fmt.Printf("Creating Jira task: %s\n", title)
		// TODO: Implement Jira API call to create task
	},
}
