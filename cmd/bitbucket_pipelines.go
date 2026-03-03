package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var pipelinesCmd = &cobra.Command{
	Use:   "pipelines",
	Short: "Pipeline operations",
	Long: `View Bitbucket Pipelines information for a repository.

Subcommands:
  list    List recent pipeline runs
  show    Show details and steps for a specific pipeline
  log     Fetch log output for a pipeline step
`,
}

// pipelinesListCmd lists recent pipelines for a repo.
var pipelinesListCmd = &cobra.Command{
	Use:     "list [repo-slug]",
	Aliases: []string{"ls"},
	Short:   "List recent pipeline runs for a repository",
	Long: `Display recent Bitbucket Pipelines runs with their status, branch, trigger, and duration.

Examples:
  devflow repo pipelines list my-repo
  devflow repo pipelines list my-repo --limit 20
  devflow repo pipelines list my-repo --json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoSlug := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		jsonOutput, _ := cmd.Flags().GetBool("json")

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

		if !jsonOutput {
			fmt.Printf("Fetching pipelines for %s/%s...\n\n", cfg.Bitbucket.Workspace, repoSlug)
		}

		pipelines, err := client.GetPipelines(repoSlug, limit)
		if err != nil {
			log.Fatalf("Error fetching pipelines: %v", err)
		}

		if len(pipelines) == 0 {
			if jsonOutput {
				fmt.Println("[]")
			} else {
				fmt.Println("No pipelines found.")
			}
			return
		}

		if jsonOutput {
			jsonBytes, err := json.MarshalIndent(pipelines, "", "  ")
			if err != nil {
				log.Fatalf("Error marshaling JSON: %v", err)
			}
			fmt.Println(string(jsonBytes))
			return
		}

		fmt.Printf("%-6s  %-12s  %-28s  %-12s  %-10s  %s\n",
			"#", "Status", "Branch / Ref", "Trigger", "Duration", "Created")
		fmt.Println(repeat("-", 100))

		for _, p := range pipelines {
			status := pipelineStateLabel(p)
			ref := p.Target.RefName
			if ref == "" {
				if p.Target.Commit != nil {
					hash := p.Target.Commit.Hash
					if len(hash) > 12 {
						hash = hash[:12]
					}
					ref = hash
				}
			}
			trigger := p.Trigger.Name
			dur := pipelineDuration(p)
			created := p.CreatedOn
			if len(created) > 19 {
				created = created[:19]
			}
			fmt.Printf("%-6d  %-12s  %-28s  %-12s  %-10s  %s\n",
				p.BuildNumber, status, truncate(ref, 28), truncate(trigger, 12), dur, created)
		}
	},
}

// pipelinesShowCmd shows details and steps for a specific pipeline.
var pipelinesShowCmd = &cobra.Command{
	Use:   "show [repo-slug] [pipeline-uuid-or-build-number]",
	Short: "Show details and steps for a specific pipeline",
	Long: `Display detailed information about a single pipeline run, including all steps and their statuses.

The pipeline can be identified by its UUID (e.g. {abc-123}) or build number.

Examples:
  devflow repo pipelines show my-repo 42
  devflow repo pipelines show my-repo {abc-123-def}
  devflow repo pipelines show my-repo 42 --json`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		repoSlug := args[0]
		pipelineRef := args[1]
		jsonOutput, _ := cmd.Flags().GetBool("json")

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

		// If the user passed a plain build number, resolve to a UUID by listing
		// pipelines and finding the matching one.
		pipelineUUID, err := resolvePipelineUUID(client, repoSlug, pipelineRef)
		if err != nil {
			log.Fatalf("Error resolving pipeline: %v", err)
		}

		pipeline, err := client.GetPipeline(repoSlug, pipelineUUID)
		if err != nil {
			log.Fatalf("Error fetching pipeline: %v", err)
		}

		steps, err := client.GetPipelineSteps(repoSlug, pipelineUUID)
		if err != nil {
			log.Fatalf("Error fetching pipeline steps: %v", err)
		}

		if jsonOutput {
			output := struct {
				Pipeline *bitbucket.Pipeline      `json:"pipeline"`
				Steps    []bitbucket.PipelineStep `json:"steps"`
			}{
				Pipeline: pipeline,
				Steps:    steps,
			}
			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				log.Fatalf("Error marshaling JSON: %v", err)
			}
			fmt.Println(string(jsonBytes))
			return
		}

		// Human-readable output
		status := pipelineStateLabel(*pipeline)
		ref := pipeline.Target.RefName
		if ref == "" && pipeline.Target.Commit != nil {
			ref = pipeline.Target.Commit.Hash
		}
		created := pipeline.CreatedOn
		if len(created) > 19 {
			created = created[:19]
		}
		completed := pipeline.CompletedOn
		if len(completed) > 19 {
			completed = completed[:19]
		}

		fmt.Printf("Pipeline #%d  %s\n", pipeline.BuildNumber, status)
		fmt.Printf("  UUID:      %s\n", pipeline.UUID)
		fmt.Printf("  Branch:    %s\n", ref)
		fmt.Printf("  Trigger:   %s\n", pipeline.Trigger.Name)
		if pipeline.Creator != nil {
			fmt.Printf("  Creator:   %s\n", pipeline.Creator.DisplayName)
		}
		fmt.Printf("  Created:   %s\n", created)
		if completed != "" {
			fmt.Printf("  Completed: %s\n", completed)
		}
		fmt.Printf("  Duration:  %s\n", pipelineDuration(*pipeline))
		fmt.Println()

		if len(steps) == 0 {
			fmt.Println("No steps found.")
			return
		}

		fmt.Printf("Steps (%d):\n", len(steps))
		fmt.Println(repeat("-", 70))
		for i, step := range steps {
			stepStatus := stepStateLabel(step)
			stepName := step.Name
			if stepName == "" {
				stepName = fmt.Sprintf("step-%d", i+1)
			}
			dur := stepDuration(step)
			fmt.Printf("  %d. %-35s  %-12s  %s\n", i+1, truncate(stepName, 35), stepStatus, dur)
		}
		fmt.Println()
		fmt.Println("To view a step's log:")
		fmt.Printf("  devflow repo pipelines log %s %s <step-uuid>\n", repoSlug, pipeline.UUID)
	},
}

// pipelinesLogCmd fetches and prints the log for a specific step.
var pipelinesLogCmd = &cobra.Command{
	Use:   "log [repo-slug] [pipeline-uuid] [step-uuid]",
	Short: "Fetch log output for a pipeline step",
	Long: `Retrieve and display the console log for a specific step in a pipeline.

The pipeline UUID and step UUID can be obtained from the 'show' subcommand.

Examples:
  devflow repo pipelines log my-repo {pipeline-uuid} {step-uuid}`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		repoSlug := args[0]
		pipelineUUID := args[1]
		stepUUID := args[2]

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

		logOutput, err := client.GetPipelineStepLog(repoSlug, pipelineUUID, stepUUID)
		if err != nil {
			log.Fatalf("Error fetching step log: %v", err)
		}

		fmt.Print(logOutput)
	},
}

func init() {
	pipelinesCmd.AddCommand(pipelinesListCmd)
	pipelinesCmd.AddCommand(pipelinesShowCmd)
	pipelinesCmd.AddCommand(pipelinesLogCmd)

	pipelinesListCmd.Flags().Int("limit", 10, "Maximum number of pipelines to return")
	pipelinesListCmd.Flags().Bool("json", false, "Output in JSON format")
	pipelinesShowCmd.Flags().Bool("json", false, "Output in JSON format")
}

// resolvePipelineUUID accepts either a UUID (contains "{" or "-") or a build
// number string and returns the pipeline UUID.
func resolvePipelineUUID(client *bitbucket.Client, repoSlug, ref string) (string, error) {
	// If it looks like a UUID (contains braces or multiple hyphens), use it directly.
	if len(ref) > 10 && (ref[0] == '{' || countRune(ref, '-') >= 4) {
		return ref, nil
	}

	// Try to parse as a build number.
	buildNum, err := strconv.Atoi(ref)
	if err != nil {
		// Not numeric either - treat as UUID.
		return ref, nil
	}

	// Fetch recent pipelines and match by build number.
	pipelines, err := client.GetPipelines(repoSlug, 100)
	if err != nil {
		return "", fmt.Errorf("fetching pipelines to resolve build number: %w", err)
	}

	for _, p := range pipelines {
		if p.BuildNumber == buildNum {
			return p.UUID, nil
		}
	}

	return "", fmt.Errorf("no pipeline found with build number %d", buildNum)
}

func countRune(s string, r rune) int {
	count := 0
	for _, c := range s {
		if c == r {
			count++
		}
	}
	return count
}

// pipelineStateLabel returns a human-readable status label with icon.
func pipelineStateLabel(p bitbucket.Pipeline) string {
	name := p.State.Name
	switch name {
	case "COMPLETED":
		if p.State.Result != nil {
			return pipelineResultIcon(p.State.Result.Name) + " " + p.State.Result.Name
		}
		return "✅ COMPLETED"
	case "IN_PROGRESS":
		stage := ""
		if p.State.Stage != nil {
			stage = p.State.Stage.Name
		}
		if stage != "" {
			return "🔄 " + stage
		}
		return "🔄 IN_PROGRESS"
	case "PENDING":
		return "⏳ PENDING"
	case "PAUSED":
		return "⏸  PAUSED"
	case "HALTED":
		return "🛑 HALTED"
	default:
		if name == "" {
			return "❓ UNKNOWN"
		}
		return "📝 " + name
	}
}

func pipelineResultIcon(result string) string {
	switch result {
	case "SUCCESSFUL":
		return "✅"
	case "FAILED":
		return "❌"
	case "ERROR":
		return "💥"
	case "STOPPED":
		return "🛑"
	default:
		return "📝"
	}
}

// stepStateLabel returns a human-readable label for a pipeline step.
func stepStateLabel(step bitbucket.PipelineStep) string {
	name := step.State.Name
	switch name {
	case "COMPLETED":
		if step.State.Result != nil {
			return pipelineResultIcon(step.State.Result.Name) + " " + step.State.Result.Name
		}
		return "✅ COMPLETED"
	case "IN_PROGRESS":
		return "🔄 RUNNING"
	case "PENDING":
		return "⏳ PENDING"
	case "PAUSED":
		return "⏸  PAUSED"
	case "HALTED":
		return "🛑 HALTED"
	default:
		if name == "" {
			return "❓ UNKNOWN"
		}
		return name
	}
}

// pipelineDuration returns a human-readable duration string for a pipeline.
func pipelineDuration(p bitbucket.Pipeline) string {
	secs := p.BuildSecondsUsed
	if secs <= 0 {
		return "-"
	}
	return formatSeconds(secs)
}

// stepDuration returns a human-readable duration for a step.
func stepDuration(step bitbucket.PipelineStep) string {
	secs := step.DurationInSeconds
	if secs <= 0 {
		return "-"
	}
	return formatSeconds(secs)
}

func formatSeconds(secs int) string {
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	mins := secs / 60
	remaining := secs % 60
	if remaining == 0 {
		return fmt.Sprintf("%dm", mins)
	}
	return fmt.Sprintf("%dm%ds", mins, remaining)
}


