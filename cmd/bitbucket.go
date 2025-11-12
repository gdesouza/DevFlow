package cmd

import (
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Repository operations",
	Long: `Manage Bitbucket repositories.

Subcommands:
  list        List workspace repositories (pagination & interactive)
  search      Regex search repositories by name (and optional description)
  show        Show detailed information about a repository
  remotes     Show HTTPS/SSH clone URLs for a repository
  readme      Display repository README contents (tries common filenames)
  watch       Manage watched repositories (add/remove/toggle/list)
`,
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
	repoCmd.AddCommand(showRepoCmd)
	repoCmd.AddCommand(remotesCmd)
	repoCmd.AddCommand(readmeCmd)
	pullrequestCmd.AddCommand(listPRsCmd)
	pullrequestCmd.AddCommand(showPRCmd)
	pullrequestCmd.AddCommand(myPRsCmd)
	pullrequestCmd.AddCommand(createPRCmd)
	pullrequestCmd.AddCommand(participatingCmd)
	pullrequestCmd.AddCommand(buildsCmd)
	pullrequestCmd.AddCommand(setStatusCmd)
}
