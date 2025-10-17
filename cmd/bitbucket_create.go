package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var (
	sourceBranch      string
	destinationBranch string
	prRepoSlug        string
	prDescription     string
	prReviewers       []string
	openInBrowser     bool
)

func detectCurrentGitBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "" // silent fallback
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" { // detached
		return ""
	}
	return branch
}

var createPRCmd = &cobra.Command{
	Use:     "create [title]",
	Aliases: []string{"create-pr"},
	Short:   "Create a pull request",
	Long:    `Create a new pull request with the specified title, description, reviewers, and optional auto-detected branches`,
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

		// Use provided repo slug
		slug := prRepoSlug
		if slug == "" {
			log.Fatal("Repository slug is required. Use --repo flag")
		}

		// Auto-detect source branch if not supplied
		srcBranch := sourceBranch
		if srcBranch == "" {
			if detected := detectCurrentGitBranch(); detected != "" {
				srcBranch = detected
			} else {
				srcBranch = "develop"
			}
		}

		// Destination branch: attempt API main branch lookup if not provided
		destBranch := destinationBranch
		client := bitbucket.NewClient(&cfg.Bitbucket)
		if destBranch == "" {
			if mainBranch, err := client.GetRepositoryMainBranch(slug); err == nil && mainBranch != "" {
				destBranch = mainBranch
			} else {
				destBranch = "main"
			}
		}

		// Create pull request with description and reviewers
		pr, err := client.CreatePullRequest(slug, title, prDescription, srcBranch, destBranch, prReviewers)
		if err != nil {
			log.Fatalf("Error creating pull request: %v", err)
		}

		// Display success message
		fmt.Printf("✅ Successfully created pull request!\n")
		fmt.Printf("🔗 #%d - %s\n", pr.ID, pr.Title)
		fmt.Printf("📂 %s → %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name)
		if pr.Description != "" {
			fmt.Printf("📝 %s\n", pr.Description)
		}
		fmt.Printf("👤 Author: %s\n", pr.Author.DisplayName)
		if len(prReviewers) > 0 {
			fmt.Printf("👀 Reviewers: %s\n", strings.Join(prReviewers, ", "))
		}
		if pr.Links.HTML.Href != "" {
			fmt.Printf("🌐 URL: %s\n", pr.Links.HTML.Href)
			if openInBrowser {
				_ = exec.Command("xdg-open", pr.Links.HTML.Href).Start() // best-effort
			}
		}
	},
}

func init() {
	createPRCmd.Flags().StringVarP(&prRepoSlug, "repo", "r", "", "Repository slug (required)")
	createPRCmd.Flags().StringVarP(&sourceBranch, "source", "s", "", "Source branch (auto-detect current)")
	createPRCmd.Flags().StringVarP(&destinationBranch, "dest", "d", "", "Destination branch (auto-detect main)")
	createPRCmd.Flags().StringVarP(&prDescription, "description", "m", "", "Pull request description/body")
	createPRCmd.Flags().StringSliceVarP(&prReviewers, "reviewer", "R", []string{}, "Reviewer username (repeatable)")
	createPRCmd.Flags().BoolVarP(&openInBrowser, "open", "o", false, "Open PR in browser after creation")
	if err := createPRCmd.MarkFlagRequired("repo"); err != nil {
		panic(err)
	}
}
