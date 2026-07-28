package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "devflow",
	Short: "CLI tool for development workflow management",
	Long: `A command-line interface tool for streamlining development workflows with Jira and Bitbucket.
Perfect for developers who want to manage tasks and repositories from the terminal.`,
	PersistentPreRunE: validateFormat,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", formatDetailed, "Output format: json, tabular, or detailed")
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(tasksCmd)
	rootCmd.AddCommand(repoCmd)
	rootCmd.AddCommand(pullrequestCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(jenkinsCmd)
}
