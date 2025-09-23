package cmd

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Manage CLI configuration including API tokens and settings`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, run the setup command
		setupConfigCmd.Run(cmd, args)
	},
}

func init() {
	configCmd.AddCommand(setConfigCmd)
	configCmd.AddCommand(getConfigCmd)
	configCmd.AddCommand(setupConfigCmd)
}
