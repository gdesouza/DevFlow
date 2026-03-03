package cmd

import (
	"fmt"
	"log"
	"strconv"

	"devflow/internal/config"
	"devflow/internal/jenkins"
	"github.com/spf13/cobra"
)

var jenkinsLogsCmd = &cobra.Command{
	Use:     "logs [job-name] [build-number]",
	Aliases: []string{"log", "console"},
	Short:   "Fetch Jenkins console logs for a build",
	Long:    `Retrieve console output for a specific Jenkins build. Use --failed-step to scope to the failing stage.`,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		jobName := args[0]
		buildNumberStr := args[1]
		failedStep, _ := cmd.Flags().GetBool("failed-step")

		buildNumber, err := strconv.Atoi(buildNumberStr)
		if err != nil {
			log.Fatalf("Invalid build number: %s", buildNumberStr)
		}

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		// Validate required config
		if cfg.Jenkins.URL == "" {
			log.Fatal("Jenkins URL not configured. Run: devflow config set jenkins.url <url>")
		}

		// Create Jenkins client
		client := jenkins.NewClient(&cfg.Jenkins)

		var logs string
		if failedStep {
			// Get logs for failed step only
			logs, err = client.GetFailedStepLog(jobName, buildNumber)
			if err != nil {
				log.Fatalf("Error fetching failed step logs: %v", err)
			}
		} else {
			// Get full console log
			logs, err = client.GetBuildLog(jobName, buildNumber)
			if err != nil {
				log.Fatalf("Error fetching build logs: %v", err)
			}
		}

		// Output the logs directly
		fmt.Print(logs)
	},
}

func init() {
	jenkinsLogsCmd.Flags().Bool("failed-step", false, "Show only the logs for the failing stage")
}
