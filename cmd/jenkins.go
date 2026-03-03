package cmd

import (
	"github.com/spf13/cobra"
)

var jenkinsCmd = &cobra.Command{
	Use:   "jenkins",
	Short: "Jenkins CI operations",
	Long: `Manage and monitor Jenkins builds and jobs.

Subcommands:
  builds      List recent builds for a job
  logs        Fetch console logs for a build
`,
}

func init() {
	jenkinsCmd.AddCommand(jenkinsBuildsCmd)
	jenkinsCmd.AddCommand(jenkinsLogsCmd)
}
