package cmd

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Manage CLI configuration including API tokens and settings`,
}

func init() {
	configCmd.AddCommand(setConfigCmd)
	configCmd.AddCommand(getConfigCmd)
}
