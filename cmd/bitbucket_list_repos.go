package cmd

import (
	"log"

	"devflow/internal/bitbucket"
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
		cfg, err := loadConfig()
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

		if wantsJSON(cmd) {
			repos, totalCount, err := client.GetRepositoriesPaged(startPage-1, pageSize)
			if err != nil {
				log.Fatalf("Error fetching repositories: %v", err)
			}
			if err := printJSON(map[string]any{
				"workspace":    cfg.Bitbucket.Workspace,
				"page":         startPage,
				"page_size":    pageSize,
				"total_count":  totalCount,
				"repositories": repos,
			}); err != nil {
				log.Fatalf("Error encoding JSON: %v", err)
			}
			return
		}
		if wantsTabular(cmd) {
			repos, _, err := client.GetRepositoriesPaged(startPage-1, pageSize)
			if err != nil {
				log.Fatalf("Error fetching repositories: %v", err)
			}
			rows := make([][]any, 0, len(repos))
			for _, repo := range repos {
				rows = append(rows, []any{repo.Name, repo.FullName, repo.Language, repo.UpdatedOn, "https://bitbucket.org/" + repo.FullName})
			}
			renderTable([]string{"Name", "Full Name", "Language", "Updated", "URL"}, rows)
			return
		}

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
