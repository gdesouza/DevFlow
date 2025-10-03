package cmd

import (
	"fmt"
	"log"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var (
	sourceBranch      string
	destinationBranch string
	prRepoSlug        string
)

var createPRCmd = &cobra.Command{
	Use:     "create [title]",
	Aliases: []string{"create-pr"},
	Short:   "Create a pull request",
	Long:    `Create a new pull request with the specified title`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		title := args[0]

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		// Validate required config
		if cfg.Bitbucket.Workspace == "" {
			log.Fatal("Bitbucket workspace not configured. Run: devflow config set bitbucket.workspace <workspace>")
		}
		if cfg.Bitbucket.Username == "" {
			log.Fatal("Bitbucket username not configured. Run: devflow config set bitbucket.username <username>")
		}
		if cfg.Bitbucket.Token == "" {
			log.Fatal("Bitbucket token not configured. Run: devflow config set bitbucket.token <token>")
		}

		// Use provided repo slug or default
		slug := prRepoSlug
		if slug == "" {
			log.Fatal("Repository slug is required. Use --repo flag")
		}

		// Set default branches if not provided
		srcBranch := sourceBranch
		if srcBranch == "" {
			srcBranch = "develop" // or could detect current branch
		}

		destBranch := destinationBranch
		if destBranch == "" {
			destBranch = "main"
		}

		// Create Bitbucket client
		client := bitbucket.NewClient(&cfg.Bitbucket)

		// Create pull request
		pr, err := client.CreatePullRequest(slug, title, srcBranch, destBranch)
		if err != nil {
			log.Fatalf("Error creating pull request: %v", err)
		}

		// Display success message
		fmt.Printf("✅ Successfully created pull request!\n")
		fmt.Printf("🔗 #%d - %s\n", pr.ID, pr.Title)
		fmt.Printf("📂 %s → %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name)
		fmt.Printf("👤 Author: %s\n", pr.Author.DisplayName)
	},
}

func init() {
	createPRCmd.Flags().StringVarP(&prRepoSlug, "repo", "r", "", "Repository slug (required)")
	createPRCmd.Flags().StringVarP(&sourceBranch, "source", "s", "", "Source branch (default: develop)")
	createPRCmd.Flags().StringVarP(&destinationBranch, "dest", "d", "", "Destination branch (default: main)")
	if err := createPRCmd.MarkFlagRequired("repo"); err != nil {
		panic(err)
	}
}
