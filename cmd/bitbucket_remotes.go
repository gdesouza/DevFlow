package cmd

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var (
	remoteSSH   bool
	remoteHTTPS bool
)

var remotesCmd = &cobra.Command{
	Use:   "remotes <repo-slug>",
	Short: "Show SSH and/or HTTPS clone endpoints",
	Long: `Display HTTPS and SSH clone URLs for a Bitbucket repository in the configured workspace.

Examples:
  # Show both
  devflow repo remotes my-service

  # Only SSH (prints just the URL)
  devflow repo remotes --ssh my-service

  # Only HTTPS (prints just the URL)
  devflow repo remotes --https my-service

Slug Tips:
  - Usually the repository slug is the lowercase name with spaces replaced by '-'
  - If unsure, run: devflow repo list | grep <name>
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if remoteSSH && remoteHTTPS {
			log.Fatal("--ssh and --https are mutually exclusive; specify only one")
		}

		repoSlug := deriveSlug(args[0])

		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
		if cfg.Bitbucket.Workspace == "" {
			log.Fatal("Bitbucket workspace not configured. Run: devflow config set bitbucket.workspace <workspace>")
		}

		workspace := cfg.Bitbucket.Workspace

		httpsURL := fmt.Sprintf("https://bitbucket.org/%s/%s.git", workspace, repoSlug)
		sshURL := fmt.Sprintf("git@bitbucket.org:%s/%s.git", workspace, repoSlug)

		// Single output mode
		if remoteSSH {
			fmt.Println(sshURL)
			return
		}
		if remoteHTTPS {
			fmt.Println(httpsURL)
			return
		}

		// Default: show both with labels
		fmt.Printf("Repository: %s/%s\n", workspace, repoSlug)
		fmt.Println("HTTPS:")
		fmt.Printf("  %s\n", httpsURL)
		fmt.Println("SSH:")
		fmt.Printf("  %s\n", sshURL)
	},
}

// deriveSlug attempts a light normalization: lowercase, spaces -> '-', trim.
// It does NOT guarantee correctness for all characters but helps common usage.
func deriveSlug(input string) string {
	s := strings.TrimSpace(input)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// Collapse consecutive dashes
	regDash := regexp.MustCompile(`-+`)
	s = regDash.ReplaceAllString(s, "-")
	return s
}

func init() {
	// Added in bitbucket.go
	remotesCmd.Flags().BoolVar(&remoteSSH, "ssh", false, "Output only the SSH clone URL")
	remotesCmd.Flags().BoolVar(&remoteHTTPS, "https", false, "Output only the HTTPS clone URL")
}
