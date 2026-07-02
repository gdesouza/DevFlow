package cmd

import (
	"fmt"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check Bitbucket authentication status",
	Long:  `Verify the configured Bitbucket credentials against the Bitbucket API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if cfg.Bitbucket.Token == "" {
			return fmt.Errorf("bitbucket token not configured. Run: devflow config set bitbucket.token <token>")
		}

		client := bitbucket.NewClient(&cfg.Bitbucket)
		if err := client.TestAuth(); err != nil {
			return fmt.Errorf("bitbucket authentication failed: %w", err)
		}

		if cfg.Bitbucket.Workspace != "" {
			fmt.Printf("Bitbucket authentication is valid for workspace %q\n", cfg.Bitbucket.Workspace)
			return nil
		}

		fmt.Println("Bitbucket authentication is valid")
		return nil
	},
}
