package cmd

import (
	"github.com/spf13/cobra"
)

var bitbucketCmd = &cobra.Command{
	Use:   "bitbucket",
	Short: "Bitbucket operations",
	Long:  `Manage Bitbucket repositories, pull requests, and pipelines`,
}

func init() {
	bitbucketCmd.AddCommand(listReposCmd)
	bitbucketCmd.AddCommand(listPRsCmd)
	bitbucketCmd.AddCommand(showPRCmd)
	bitbucketCmd.AddCommand(myPRsCmd)
	bitbucketCmd.AddCommand(createPRCmd)
	bitbucketCmd.AddCommand(testAuthCmd)
}
