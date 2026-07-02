package cmd

import (
	"log"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var (
	pageSize    int
	startPage   int
	interactive bool
)

var listReposCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"list-repos"},
	Short:   "List repositories in the workspace",
	Long:    `List repositories in the configured Bitbucket workspace with pagination support`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		if cfg.Bitbucket.Workspace == "" {
			log.Fatal("Bitbucket workspace not configured. Run: devflow config set bitbucket.workspace <workspace>")
		}
		if cfg.Bitbucket.Username == "" {
			log.Fatal("Bitbucket username not configured. Run: devflow config set bitbucket.username <username>")
		}
		if cfg.Bitbucket.Token == "" {
			log.Fatal("Bitbucket token not configured. Run: devflow config set bitbucket.token <token>")
		}

		client := bitbucket.NewClient(&cfg.Bitbucket)

		if interactive {
			runInteractiveMode(client, cfg.Bitbucket.Workspace)
			return
		}
		if startPage < 1 {
			log.Fatal("Page numbers are 1-based. Use --page 1 or higher.")
		}
		runPagedMode(client, cfg.Bitbucket.Workspace, startPage-1, pageSize)
	},
}

func init() {
	listReposCmd.Flags().IntVarP(&pageSize, "size", "s", 20, "Number of repositories per page")
	listReposCmd.Flags().IntVarP(&startPage, "page", "p", 1, "Page number to display (1-based)")
	listReposCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Enable interactive navigation mode")
}
