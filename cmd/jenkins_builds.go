package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"devflow/internal/config"
	"devflow/internal/jenkins"
	"github.com/spf13/cobra"
)

var jenkinsBuildsCmd = &cobra.Command{
	Use:     "builds [job-name]",
	Aliases: []string{"list-builds"},
	Short:   "List recent Jenkins builds for a job",
	Long:    `Display recent Jenkins build runs with their status and build numbers.`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jobName := args[0]
		limit, _ := cmd.Flags().GetInt("limit")

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

		// Get builds
		builds, err := client.GetJobBuilds(jobName, limit)
		if err != nil {
			log.Fatalf("Error fetching builds: %v", err)
		}

		if len(builds) == 0 {
			fmt.Printf("No builds found for job: %s\n", jobName)
			return
		}

		// Display builds
		fmt.Printf("🔨 Recent builds for job: %s (%d total)\n", jobName, len(builds))
		fmt.Println("=" + repeat("=", 79))
		fmt.Printf("%-8s %-12s %-20s %-12s\n", "Build #", "Status", "Started", "Duration")
		fmt.Println(repeat("-", 80))

		for _, build := range builds {
			status := formatBuildStatus(build.Result, build.Building)
			started := formatTimestamp(build.Timestamp)
			duration := formatDuration(build.Duration)

			fmt.Printf("%-8d %-12s %-20s %-12s\n",
				build.Number,
				status,
				started,
				duration,
			)
		}
	},
}

func init() {
	jenkinsBuildsCmd.Flags().IntP("limit", "l", 10, "Number of builds to retrieve")
}

func formatBuildStatus(result string, building bool) string {
	if building {
		return "🔄 BUILDING"
	}
	switch result {
	case "SUCCESS":
		return "✅ SUCCESS"
	case "FAILURE":
		return "❌ FAILURE"
	case "UNSTABLE":
		return "⚠️  UNSTABLE"
	case "ABORTED":
		return "🛑 ABORTED"
	default:
		return "❓ " + result
	}
}

func formatTimestamp(timestamp int64) string {
	t := time.Unix(timestamp/1000, 0)
	return t.Format("2006-01-02 15:04:05")
}

func formatDuration(duration int64) string {
	d := time.Duration(duration) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

func repeat(s string, n int) string {
	return strings.Repeat(s, n)
}
