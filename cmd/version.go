package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "v1.3.0" // Released version

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
