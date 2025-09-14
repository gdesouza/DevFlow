package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var (
	pageSize    int
	startPage   int
	interactive bool
)

var listReposCmd = &cobra.Command{
	Use:   "list-repos",
	Short: "List repositories in the workspace",
	Long:  `List repositories in the configured Bitbucket workspace with pagination support`,
	Run: func(cmd *cobra.Command, args []string) {
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

		if interactive {
			runInteractiveMode(client, cfg.Bitbucket.Workspace)
		} else {
			runPagedMode(client, cfg.Bitbucket.Workspace, startPage, pageSize)
		}
	},
}

func runPagedMode(client *bitbucket.Client, workspace string, page, size int) {
	// Get repositories for specific page
	repos, totalCount, err := client.GetRepositoriesPaged(page, size)
	if err != nil {
		log.Fatalf("Error fetching repositories: %v", err)
	}

	if len(repos) == 0 {
		fmt.Printf("No repositories found in workspace '%s'.\n", workspace)
		return
	}

	totalPages := (totalCount + size - 1) / size // Ceiling division
	fmt.Printf("Found %d repositories in workspace '%s' (Page %d/%d):\n\n", totalCount, workspace, page+1, totalPages)

	displayReposPage(repos, workspace)

	if totalPages > 1 {
		fmt.Printf("\n--- Page %d of %d ---\n", page+1, totalPages)
		fmt.Printf("Use --page to view other pages, or --interactive for navigation\n")
	}
}

func runInteractiveMode(client *bitbucket.Client, workspace string) {
	reader := bufio.NewReader(os.Stdin)
	currentPage := 0

	// Use the pageSize variable from flags, or ask user if not specified
	interactivePageSize := pageSize
	if pageSize == 20 { // Default value, ask user
		fmt.Print("Enter page size (default 10): ")
		sizeInput, _ := reader.ReadString('\n')
		sizeInput = strings.TrimSpace(sizeInput)
		if sizeInput == "" {
			interactivePageSize = 10
		} else if size, err := strconv.Atoi(sizeInput); err == nil && size > 0 && size <= 100 {
			interactivePageSize = size
		} else {
			fmt.Println("Invalid page size, using default of 10.")
			interactivePageSize = 10
		}
	}

	for {
		// Get repositories for current page
		repos, totalCount, err := client.GetRepositoriesPaged(currentPage, interactivePageSize)
		if err != nil {
			log.Fatalf("Error fetching repositories: %v", err)
		}

		if len(repos) == 0 && currentPage == 0 {
			fmt.Printf("No repositories found in workspace '%s'.\n", workspace)
			return
		}

		// Clear screen (ANSI escape code)
		fmt.Print("\033[2J\033[1;1H")

		totalPages := (totalCount + interactivePageSize - 1) / interactivePageSize // Ceiling division
		fmt.Printf("Found %d repositories in workspace '%s' (Page %d/%d):\n\n", totalCount, workspace, currentPage+1, totalPages)

		displayReposPage(repos, workspace)

		if totalPages <= 1 {
			fmt.Println("\n--- End of results ---")
			return
		}

		fmt.Printf("\n--- Page %d of %d ---\n", currentPage+1, totalPages)
		fmt.Print("Navigation: (n)ext, (p)revious, (g)o to page, (q)uit: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading input: %v", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "n", "next":
			if currentPage < totalPages-1 {
				currentPage++
			} else {
				fmt.Println("Already at the last page.")
				fmt.Print("Press Enter to continue...")
				reader.ReadString('\n')
			}
		case "p", "prev", "previous":
			if currentPage > 0 {
				currentPage--
			} else {
				fmt.Println("Already at the first page.")
				fmt.Print("Press Enter to continue...")
				reader.ReadString('\n')
			}
		case "g", "go":
			fmt.Printf("Enter page number (1-%d): ", totalPages)
			pageInput, _ := reader.ReadString('\n')
			pageInput = strings.TrimSpace(pageInput)
			if pageNum, err := strconv.Atoi(pageInput); err == nil && pageNum >= 1 && pageNum <= totalPages {
				currentPage = pageNum - 1
			} else {
				fmt.Printf("Invalid page number. Press Enter to continue...")
				reader.ReadString('\n')
			}
		case "q", "quit", "exit":
			return
		default:
			fmt.Printf("Invalid command. Press Enter to continue...")
			reader.ReadString('\n')
		}
	}
}

func displayReposPage(repos []bitbucket.Repository, workspace string) {
	for _, repo := range repos {
		// Privacy indicator
		privacyIcon := "🔓"
		if repo.IsPrivate {
			privacyIcon = "🔒"
		}

		// Minimal format: name, URL, and language only
		fmt.Printf("%s %s 🔗 https://bitbucket.org/%s/%s", privacyIcon, repo.Name, workspace, repo.Name)

		if repo.Language != "" {
			fmt.Printf(" 💻 %s", repo.Language)
		}

		fmt.Println()
	}
}

func init() {
	listReposCmd.Flags().IntVarP(&pageSize, "size", "s", 20, "Number of repositories per page")
	listReposCmd.Flags().IntVarP(&startPage, "page", "p", 1, "Page number to display (1-based)")
	listReposCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Enable interactive navigation mode")
}
