package cmd

import (
	"fmt"
	"log"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var readmeRaw bool

var readmeCmd = &cobra.Command{
	Use:   "readme [repo-slug]",
	Short: "Show repository README contents",
	Long:  "Fetch and display the README for a Bitbucket repository if it exists (tries common filenames).",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoSlug := args[0]

		cfg, err := config.Load()
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
		filename, contents, err := client.GetRepositoryReadme(repoSlug)
		if err != nil {
			log.Fatalf("%v", err)
		}

		if readmeRaw {
			fmt.Print(contents)
			return
		}

		fmt.Printf("📄 README (%s)\n", filename)
		fmt.Println("────────────────────────────────────────────────────────────────")
		fmt.Println(contents)
	},
}

func init() {
	readmeCmd.Flags().BoolVar(&readmeRaw, "raw", false, "Output raw README contents only")
}
