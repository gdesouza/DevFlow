package cmd

import (
	"fmt"
	"log"
	"time"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var showRepoCmd = &cobra.Command{
	Use:   "show <repo-name>",
	Short: "Show detailed information about a repository",
	Long:  `Display detailed information about a Bitbucket repository including description, size, language, and timestamps.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoSlug := args[0]

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

		// Create Bitbucket client
		client := bitbucket.NewClient(&cfg.Bitbucket)

		// Get repository details
		repo, err := client.GetRepository(repoSlug)
		if err != nil {
			log.Fatalf("Error fetching repository details: %v", err)
		}

		displayRepositoryDetails(repo, cfg.Bitbucket.Workspace)
	},
}

func displayRepositoryDetails(repo *bitbucket.Repository, workspace string) {
	// Header
	fmt.Printf("📁 Repository: %s\n", repo.Name)
	fmt.Printf("🌐 Full Name: %s\n", repo.FullName)
	if repo.Description != "" {
		fmt.Printf("📝 Description: %s\n", repo.Description)
	} else {
		fmt.Printf("📝 Description: (no description)\n")
	}
	fmt.Println()

	// Visibility and Privacy
	privacyIcon := "🔓"
	privacyText := "Public"
	if repo.IsPrivate {
		privacyIcon = "🔒"
		privacyText = "Private"
	}
	fmt.Printf("%s Visibility: %s\n", privacyIcon, privacyText)

	// Programming Language
	if repo.Language != "" {
		fmt.Printf("💻 Language: %s\n", repo.Language)
	} else {
		fmt.Printf("💻 Language: (not detected)\n")
	}

	// Main branch
	if repo.MainBranch.Name != "" {
		fmt.Printf("🌿 Default Branch: %s\n", repo.MainBranch.Name)
	}

	// Repository size
	if repo.Size > 0 {
		fmt.Printf("📏 Size: %s\n", formatSize(repo.Size))
	}

	fmt.Println()

	// Timestamps
	if repo.CreatedOn != "" {
		if createdTime, err := time.Parse(time.RFC3339, repo.CreatedOn); err == nil {
			fmt.Printf("📅 Created: %s (%s)\n", createdTime.Format("2006-01-02 15:04:05"), formatRelativeTime(createdTime))
		} else {
			fmt.Printf("📅 Created: %s\n", repo.CreatedOn)
		}
	}

	if repo.UpdatedOn != "" {
		if updatedTime, err := time.Parse(time.RFC3339, repo.UpdatedOn); err == nil {
			fmt.Printf("🕒 Last Updated: %s (%s)\n", updatedTime.Format("2006-01-02 15:04:05"), formatRelativeTime(updatedTime))
		} else {
			fmt.Printf("🕒 Last Updated: %s\n", repo.UpdatedOn)
		}
	}

	fmt.Println()

	// Links
	fmt.Printf("🔗 Repository URL: https://bitbucket.org/%s\n", repo.FullName)
	fmt.Printf("📋 Clone (HTTPS): https://bitbucket.org/%s.git\n", repo.FullName)
	fmt.Printf("📋 Clone (SSH): git@bitbucket.org:%s.git\n", repo.FullName)
}

// formatSize converts bytes to a human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatRelativeTime returns a human-readable relative time string
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 30*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	} else {
		years := int(diff.Hours() / (24 * 365))
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}
