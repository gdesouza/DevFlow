package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev" // Default version for development builds

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of devflow",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
