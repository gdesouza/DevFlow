package cmd

import (
	"fmt"
	"io"
	"os"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

type authChecker interface {
	TestAuth() error
}

func runAuthStatus(loadConfig func() (*config.Config, error), newClient func(*config.BitbucketConfig) authChecker, out io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.Bitbucket.Token == "" {
		return fmt.Errorf("bitbucket token not configured. Run: devflow config set bitbucket.token <token>")
	}

	client := newClient(&cfg.Bitbucket)
	if err := client.TestAuth(); err != nil {
		return fmt.Errorf("bitbucket authentication failed: %w", err)
	}

	if cfg.Bitbucket.Workspace != "" {
		_, _ = fmt.Fprintf(out, "Bitbucket authentication is valid for workspace %q\n", cfg.Bitbucket.Workspace)
		return nil
	}

	_, _ = fmt.Fprintln(out, "Bitbucket authentication is valid")
	return nil
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check Bitbucket authentication status",
	Long:  `Verify the configured Bitbucket credentials against the Bitbucket API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuthStatus(config.Load, func(cfg *config.BitbucketConfig) authChecker {
			return bitbucket.NewClient(cfg)
		}, os.Stdout)
	},
}
