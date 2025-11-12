package cmd

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var (
	stateAllowed = []string{"SUCCESSFUL", "FAILED", "INPROGRESS", "STOPPED", "ERROR", "PENDING", "CANCELLED"}
)

var setStatusCmd = &cobra.Command{
	Use:     "set-status [repo-slug] [commit-hash]",
	Aliases: []string{"set-build", "status-set"},
	Short:   "Create or update a build/status for a commit",
	Long: `Create or update a Bitbucket commit status (build/deployment/check).

Minimal usage (auto-fill details):
  devflow pullrequest set-status my-repo <commit-sha> --state SUCCESSFUL --key ci/pipeline --description "All tests passed"

If a status with the given key already exists for the commit and you omit --name / --url / --description, existing values are reused. When creating a new status and --name is omitted it defaults to the key.

States: SUCCESSFUL, FAILED, INPROGRESS, STOPPED, ERROR, PENDING, CANCELLED
Reusing the same --key upserts (updates) the existing status.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		repoSlug := args[0]
		commitHash := args[1]

		state, _ := cmd.Flags().GetString("state")
		key, _ := cmd.Flags().GetString("key")
		name, _ := cmd.Flags().GetString("name")
		urlStr, _ := cmd.Flags().GetString("url")
		description, _ := cmd.Flags().GetString("description")

		if err := validateSetStatusInputs(state, key, name, urlStr); err != nil {
			log.Fatalf("Input validation error: %v", err)
		}

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

		client := bitbucket.NewClient(&cfg.Bitbucket)
		fmt.Printf("Setting status for commit %s in %s...\n", shortHash(commitHash), repoSlug)

		st, err := client.SetCommitStatus(repoSlug, commitHash, state, key, name, urlStr, description)
		if err != nil {
			log.Fatalf("Failed to set commit status: %v", err)
		}

		icon := statusStateIcon(st.State)
		fmt.Printf("%s %s - %s\n", icon, st.State, displayName(st))
		if st.Description != "" {
			fmt.Printf("  %s\n", firstLine(st.Description))
		}
		if st.URL != "" {
			fmt.Printf("  🔗 %s\n", st.URL)
		}
		fmt.Printf("  Updated: %s\n", relativeTime(st.UpdatedOn))
	},
}

func init() {
	setStatusCmd.Flags().String("state", "", "Status state (SUCCESSFUL, FAILED, INPROGRESS, STOPPED, ERROR, PENDING, CANCELLED) (required)")
	setStatusCmd.Flags().String("key", "", "Unique status key (e.g. ci/pipeline) (required)")
	setStatusCmd.Flags().String("name", "", "Human-friendly status name (defaults to key)")
	setStatusCmd.Flags().String("url", "", "Link to build or deployment details")
	setStatusCmd.Flags().String("description", "", "Short description of the status result")
	if err := setStatusCmd.MarkFlagRequired("state"); err != nil {
		panic(err)
	}
	if err := setStatusCmd.MarkFlagRequired("key"); err != nil {
		panic(err)
	}
}

func validateSetStatusInputs(state, key, name, urlStr string) error {
	if strings.TrimSpace(state) == "" {
		return errors.New("--state is required")
	}
	if strings.TrimSpace(key) == "" {
		return errors.New("--key is required")
	}
	// Validate state against allowlist
	allowed := false
	for _, s := range stateAllowed {
		if state == s {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("invalid state '%s' (allowed: %s)", state, strings.Join(stateAllowed, ", "))
	}
	if len(key) > 100 {
		return fmt.Errorf("key too long (%d > 100)", len(key))
	}
	if name != "" && len(name) > 200 {
		return fmt.Errorf("name too long (%d > 200)", len(name))
	}
	if urlStr != "" {
		if _, err := url.ParseRequestURI(urlStr); err != nil {
			return fmt.Errorf("invalid url: %v", err)
		}
	}
	return nil
}

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

func displayName(st *bitbucket.CommitStatus) string {
	if st.Name != "" {
		return st.Name
	}
	return st.Key
}
