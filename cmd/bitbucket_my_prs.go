package cmd

import (
	"fmt"
	"log"
	"strings"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var (
	myPRsRepoSlug string
	myPRsAllRepos bool
)

var myPRsCmd = &cobra.Command{
	Use:     "mine [repo-slug]",
	Aliases: []string{"my-prs", "my"},
	Short:   "List PRs where you are author (watched repos)",
	Long: `List pull requests you authored.

Behavior changes:
- Without --all-repos and no slug: aggregates across all watched repositories.
- With a slug: requires that slug to be watched.
- --all-repos still uses workspace-level endpoint (ignores watch list).`,
	Run: func(cmd *cobra.Command, args []string) {
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

		// For --all-repos, we need either bitbucket_user or will use username as fallback
		if myPRsAllRepos && cfg.Bitbucket.BitbucketUser == "" {
			fmt.Printf("Note: For optimal performance with --all-repos, set your Bitbucket username:\n")
			fmt.Printf("  devflow config set bitbucket.bitbucket_user <your-bitbucket-username>\n")
			fmt.Printf("Currently using email address as fallback.\n\n")
		}

		// Create Bitbucket client
		client := bitbucket.NewClient(&cfg.Bitbucket)

		if myPRsAllRepos {
			userPRs, err := getPRsToReviewFromAllRepos(client, cfg.Bitbucket.Username)
			if err != nil {
				log.Fatalf("Error fetching pull requests: %v", err)
			}
			displayPRsToReview(userPRs, "all repositories", true)
			return
		}

		// Build watched set
		watched := map[string]struct{}{}
		for _, w := range cfg.Bitbucket.WatchedRepos {
			watched[strings.ToLower(w)] = struct{}{}
		}
		if len(watched) == 0 {
			log.Fatal("No watched repositories. Add some with: devflow bitbucket repo watch add <repo>")
		}

		// If slug provided ensure watched; else aggregate across watched
		slug := myPRsRepoSlug
		if slug == "" && len(args) > 0 {
			slug = args[0]
		}

		if slug != "" {
			if _, ok := watched[strings.ToLower(slug)]; !ok {
				log.Fatalf("Repository '%s' not watched. Add with: devflow bitbucket repo watch add %s", slug, slug)
			}
			prs, err := client.GetPullRequestsWithReviewers(slug)
			if err != nil {
				log.Fatalf("Error fetching pull requests: %v", err)
			}
			userPRs := filterPRsForUser(prs, cfg.Bitbucket.Username)
			var prsWithRepo []PRWithRepo
			for _, pr := range userPRs {
				prsWithRepo = append(prsWithRepo, PRWithRepo{PR: pr, RepoSlug: slug})
			}
			displayPRsToReview(prsWithRepo, slug, false)
			return
		}

		// Aggregate across watched
		var all []PRWithRepo
		for w := range watched {
			prs, err := client.GetPullRequestsWithReviewers(w)
			if err != nil {
				fmt.Printf("Warning: failed fetching PRs for %s: %v\n", w, err)
				continue
			}
			userPRs := filterPRsForUser(prs, cfg.Bitbucket.Username)
			for _, pr := range userPRs {
				all = append(all, PRWithRepo{PR: pr, RepoSlug: w})
			}
		}
		displayPRsToReview(all, "watched repositories", true)
	},
}

func init() {
	myPRsCmd.Flags().StringVarP(&myPRsRepoSlug, "repo", "r", "", "Repository slug (required when not using --all-repos)")
	myPRsCmd.Flags().BoolVar(&myPRsAllRepos, "all-repos", false, "Search across all repositories in the workspace")
}

// PRWithRepo holds a pull request along with its repository information
type PRWithRepo struct {
	PR       bitbucket.PullRequestWithReviewers
	RepoSlug string
}

// getPRsToReviewFromAllRepos fetches PRs to review from all repositories using the efficient workspace endpoint
func getPRsToReviewFromAllRepos(client *bitbucket.Client, username string) ([]PRWithRepo, error) {
	// Load config to get workspace info
	cfg, _ := config.Load()
	workspace := cfg.Bitbucket.Workspace

	fmt.Printf("Searching for PRs to review in workspace: %s\n", workspace)
	fmt.Printf("Using efficient workspace-level API to find all PRs where you are a reviewer...\n")

	// Use the efficient workspace endpoint to get all PRs where user is reviewer
	// Use BitbucketUser if set, otherwise fall back to Username
	bitbucketUsername := cfg.Bitbucket.BitbucketUser
	if bitbucketUsername == "" {
		bitbucketUsername = cfg.Bitbucket.Username
		fmt.Printf("Warning: Using email as username. For better performance, set your Bitbucket username with:\n")
		fmt.Printf("  devflow config set bitbucket.bitbucket_user <your-username>\n")
	}

	prs, err := client.GetWorkspacePullRequestsForUser(bitbucketUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace PRs for user: %w", err)
	}

	fmt.Printf("Found %d PRs where you are assigned as reviewer across the workspace\n", len(prs))

	// Convert to PRWithRepo format (we need to extract repo slug from PR data)
	var prsWithRepo []PRWithRepo
	for _, pr := range prs {
		// For workspace-level PRs, we need to extract the repository from the links or other data
		// The PR structure should contain repository information
		repoSlug := extractRepoSlugFromPR(pr)
		if repoSlug == "" {
			fmt.Printf("Warning: Could not extract repository slug for PR #%d\n", pr.ID)
			continue
		}

		prsWithRepo = append(prsWithRepo, PRWithRepo{
			PR:       pr,
			RepoSlug: repoSlug,
		})
	}

	fmt.Printf("Completed search in workspace: %s\n", workspace)
	return prsWithRepo, nil
}

// extractRepoSlugFromPR extracts the repository slug from PR data
// This is a helper function to get the repo slug from the workspace-level PR response
func extractRepoSlugFromPR(pr bitbucket.PullRequestWithReviewers) string {
	// Use the repository name from the source branch (should be the same for destination)
	if pr.Source.Repository.Name != "" {
		return pr.Source.Repository.Name
	}

	// Fallback to destination if source doesn't have it
	if pr.Destination.Repository.Name != "" {
		return pr.Destination.Repository.Name
	}

	// If neither has repository info, return unknown
	return "unknown-repo"
}

// displayPRsToReview displays the PRs to review
func displayPRsToReview(prs []PRWithRepo, source string, showRepo bool) {
	if len(prs) == 0 {
		fmt.Printf("No pull requests found for %s where you are assigned as reviewer.\n", source)
		return
	}

	fmt.Printf("Found %d pull requests for %s where you are assigned as reviewer:\n\n", len(prs), source)

	// Load config for workspace
	cfg, _ := config.Load()

	for _, prWithRepo := range prs {
		pr := prWithRepo.PR
		repoSlug := prWithRepo.RepoSlug

		statusIcon := getPRStatusIcon(pr.State)
		if showRepo {
			fmt.Printf("%s #%d - %s [%s] 🔗 https://bitbucket.org/%s/%s/pull-requests/%d\n",
				statusIcon, pr.ID, pr.Title, repoSlug, cfg.Bitbucket.Workspace, repoSlug, pr.ID)
		} else {
			fmt.Printf("%s #%d - %s 🔗 https://bitbucket.org/%s/%s/pull-requests/%d\n",
				statusIcon, pr.ID, pr.Title, cfg.Bitbucket.Workspace, repoSlug, pr.ID)
		}
		fmt.Printf("   👤 Author: %s\n", pr.Author.DisplayName)
		fmt.Printf("   📂 %s → %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name)
		fmt.Println()
	}
}

// filterPRsForUser filters pull requests where the given username is a reviewer
func filterPRsForUser(prs []bitbucket.PullRequestWithReviewers, username string) []bitbucket.PullRequestWithReviewers {
	var filtered []bitbucket.PullRequestWithReviewers

	fmt.Printf("Debug: Filtering %d PRs for user '%s'\n", len(prs), username)

	for _, pr := range prs {
		fmt.Printf("Debug: PR #%d '%s' has %d reviewers\n", pr.ID, pr.Title, len(pr.Reviewers))
		for _, reviewer := range pr.Reviewers {
			fmt.Printf("Debug:   Reviewer: '%s'\n", reviewer.DisplayName)
			// Check if the reviewer display name matches (case-insensitive)
			if strings.EqualFold(reviewer.DisplayName, username) {
				fmt.Printf("Debug:   ✓ Match found for user '%s'\n", username)
				filtered = append(filtered, pr)
				break
			}
		}
	}

	fmt.Printf("Debug: Found %d matching PRs\n", len(filtered))
	return filtered
}
