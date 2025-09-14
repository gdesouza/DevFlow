package cmd

import (
	"fmt"
	"log"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var testAuthCmd = &cobra.Command{
	Use:   "test-auth",
	Short: "Test Bitbucket authentication",
	Long:  `Test basic authentication with Bitbucket API`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		// Validate required config
		if cfg.Bitbucket.Token == "" {
			log.Fatal("Bitbucket token not configured. Run: devflow config set bitbucket.token <token>")
		}

		// Create Bitbucket client
		client := bitbucket.NewClient(&cfg.Bitbucket)

		// Test Bearer authentication first
		fmt.Println("Testing Bearer authentication...")
		err = client.TestAuth()
		if err != nil {
			fmt.Println("Bearer auth failed, trying Basic auth...")
			err = client.TestBasicAuth()
			if err != nil {
				log.Fatalf("Both authentication methods failed: %v", err)
			}
			fmt.Println("✅ Bitbucket authentication successful with Basic auth!")
		} else {
			fmt.Println("✅ Bitbucket authentication successful with Bearer auth!")
		}

		fmt.Printf("Token appears to be valid for basic API access.\n")
	},
}

func init() {
	// This will be called when the bitbucket command is initialized
}
