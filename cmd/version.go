package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev" // Development build; overridden at build via -ldflags

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of devflow",
	Run: func(cmd *cobra.Command, args []string) {
		if wantsJSON(cmd) {
			if err := printJSON(map[string]string{"version": version}); err != nil {
				fmt.Printf("Error encoding JSON: %v\n", err)
			}
			return
		}
		if wantsTabular(cmd) {
			renderKeyValueTable([][2]string{{"Version", version}})
			return
		}
		fmt.Println(version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
