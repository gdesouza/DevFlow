package cmd

import "github.com/spf13/cobra"

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication management",
	Long:  `Check and manage authentication for connected services.`,
}

func init() {
	authCmd.AddCommand(authStatusCmd)
}
