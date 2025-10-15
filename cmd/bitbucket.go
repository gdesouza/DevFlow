package cmd

import (
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Repository operations",
	Long:  `Manage Bitbucket repositories`,
}

var pullrequestCmd = &cobra.Command{
	Use:     "pullrequest",
	Aliases: []string{"pr", "pullrequests", "prs"},
	Short:   "Pull request operations",
	Long:    `Manage Bitbucket pull requests`,
}

func init() {
	repoCmd.AddCommand(listReposCmd)
	repoCmd.AddCommand(searchReposCmd)
	repoCmd.AddCommand(remotesCmd)
	pullrequestCmd.AddCommand(listPRsCmd)
	pullrequestCmd.AddCommand(showPRCmd)
	pullrequestCmd.AddCommand(myPRsCmd)
	pullrequestCmd.AddCommand(createPRCmd)
	pullrequestCmd.AddCommand(participatingCmd)
}
