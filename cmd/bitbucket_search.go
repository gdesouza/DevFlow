package cmd

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var (
	caseSensitive      bool
	includeDescription bool
)

var searchReposCmd = &cobra.Command{
	Use:   "search <regex>",
	Short: "Search repositories by regex",
	Long: `Search Bitbucket workspace repositories whose name (and optionally description) matches the provided regular expression.
Examples:
  devflow repo search 'api-.*'
  devflow repo search --case-sensitive 'API-[A-Z]+'
  devflow repo search -d 'terraform'
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pattern := args[0]

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		if cfg.Bitbucket.Workspace == "" {
			log.Fatal("Bitbucket workspace not configured. Run: devflow config set bitbucket.workspace <workspace>")
		}
		if cfg.Bitbucket.Token == "" {
			log.Fatal("Bitbucket token not configured. Run: devflow config set bitbucket.token <token>")
		}

		// Prepare regex (case-insensitive by default unless caseSensitive)
		if !caseSensitive && !strings.HasPrefix(pattern, "(?i)") {
			pattern = "(?i)" + pattern
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Fatalf("Invalid regex: %v", err)
		}

		client := bitbucket.NewClient(&cfg.Bitbucket)
		repos, err := client.GetRepositories()
		if err != nil {
			log.Fatalf("Error fetching repositories: %v", err)
		}

		var matches []bitbucket.Repository
		for _, r := range repos {
			if re.MatchString(r.Name) || (includeDescription && re.MatchString(r.Description)) {
				matches = append(matches, r)
			}
		}

		if len(matches) == 0 {
			fmt.Printf("No repositories matched regex in workspace '%s'.\n", cfg.Bitbucket.Workspace)
			return
		}

		// Sort by name for stable output
		sort.Slice(matches, func(i, j int) bool { return strings.ToLower(matches[i].Name) < strings.ToLower(matches[j].Name) })

		fmt.Printf("Found %d matching repositories in workspace '%s':\n\n", len(matches), cfg.Bitbucket.Workspace)
		displayReposPage(matches, cfg.Bitbucket.Workspace)
	},
}

func init() {
	searchReposCmd.Flags().BoolVarP(&caseSensitive, "case-sensitive", "c", false, "Enable case sensitive matching (default insensitive)")
	searchReposCmd.Flags().BoolVarP(&includeDescription, "description", "d", false, "Match against description in addition to name")
}
