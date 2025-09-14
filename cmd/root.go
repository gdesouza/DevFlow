package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "devflow",
	Short: "CLI tool for development workflow management",
	Long: `A command-line interface tool for streamlining development workflows with Jira and Bitbucket.
Perfect for developers who want to manage tasks and repositories from the terminal.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(jiraCmd)
	rootCmd.AddCommand(bitbucketCmd)
	rootCmd.AddCommand(configCmd)
}
