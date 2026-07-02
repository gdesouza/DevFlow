package cmd

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"devflow/internal/bitbucket"
	"devflow/internal/config"
	"golang.org/x/term"
)

func runPagedMode(client *bitbucket.Client, workspace string, page, size int) {
	repos, totalCount, err := client.GetRepositoriesPaged(page, size)
	if err != nil {
		log.Fatalf("Error fetching repositories: %v", err)
	}

	if len(repos) == 0 {
		fmt.Printf("No repositories found in workspace '%s'.\n", workspace)
		return
	}

	totalPages := (totalCount + size - 1) / size
	fmt.Printf("Found %d repositories in workspace '%s' (Page %d/%d):\n\n", totalCount, workspace, page+1, totalPages)

	displayReposPage(repos, workspace)

	if totalPages > 1 {
		fmt.Printf("\n--- Page %d of %d ---\n", page+1, totalPages)
		fmt.Printf("Use --page to view other pages, or --interactive for navigation\n")
	}
}

// runInteractiveMode provides an arrow-key driven UI for browsing and toggling watched repositories.
// Keys:
//
//	Up/Down: move selection
//	Left/Right: previous/next page
//	Space or Enter: toggle watch
//	w: jump to next watched repo
//	g: go to page number
//	s: save (no-op; autosave already happens)
//	q: quit
func runInteractiveMode(client *bitbucket.Client, workspace string) {
	interactivePageSize := pageSize
	if pageSize == 20 {
		fmt.Print("Enter page size (default 10): ")
		var inp string
		if _, err := fmt.Scanln(&inp); err != nil {
			inp = ""
		}
		if strings.TrimSpace(inp) == "" {
			interactivePageSize = 10
		} else if n, err := strconv.Atoi(strings.TrimSpace(inp)); err == nil && n > 0 && n <= 100 {
			interactivePageSize = n
		} else {
			fmt.Println("Invalid page size, using default of 10.")
			interactivePageSize = 10
		}
	}

	cfg, _ := config.Load()
	watchedSet := map[string]struct{}{}
	for _, w := range cfg.Bitbucket.WatchedRepos {
		watchedSet[strings.ToLower(w)] = struct{}{}
	}

	currentPage := 0
	selection := 0
	var totalPages int
	var totalCount int

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("Failed to set raw mode: %v", err)
	}
	defer func() {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}()

	stdin := os.Stdin
	buf := make([]byte, 3)

	fetchPage := func(page int) ([]bitbucket.Repository, error) {
		repos, count, err := client.GetRepositoriesPaged(page, interactivePageSize)
		if err != nil {
			return nil, err
		}
		totalCount = count
		totalPages = (totalCount + interactivePageSize - 1) / interactivePageSize
		if selection >= len(repos) {
			selection = len(repos) - 1
			if selection < 0 {
				selection = 0
			}
		}
		return repos, nil
	}

	repos, err := fetchPage(currentPage)
	if err != nil {
		log.Fatalf("Error fetching repositories: %v", err)
	}
	if len(repos) == 0 {
		fmt.Printf("No repositories found in workspace '%s'.\n", workspace)
		return
	}

	for {
		var bldr strings.Builder
		bldr.WriteString("\033[2J\033[1;1H")
		fmt.Fprintf(&bldr, "Found %d repositories in workspace '%s' (Page %d/%d)\r\n\r\n", totalCount, workspace, currentPage+1, totalPages)
		for i, r := range repos {
			privacyIcon := "PRV"
			if !r.IsPrivate {
				privacyIcon = "PUB"
			}
			watchIcon := "[ ]"
			if _, ok := watchedSet[strings.ToLower(r.Name)]; ok {
				watchIcon = "[*]"
			}
			if i == selection {
				fmt.Fprintf(&bldr, "\033[7m%3d %s %s %s\033[0m\r\n", i+1, watchIcon, privacyIcon, r.Name)
			} else {
				fmt.Fprintf(&bldr, "%3d %s %s %s\r\n", i+1, watchIcon, privacyIcon, r.Name)
			}
		}
		bldr.WriteString("\r\nKeys: ↑/↓ move  ←/→ page  Space/Enter toggle  w next-watched  g go-page  s save  q quit\r\n")
		if _, err := os.Stdout.WriteString(bldr.String()); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write to stdout: %v\n", err)
		}

		_ = stdin.SetReadDeadline(time.Now().Add(5 * time.Minute))
		n, err := stdin.Read(buf[:1])
		if err != nil || n == 0 {
			continue
		}
		b := buf[0]

		if b == 3 {
			return
		}

		if b == 27 {
			_, _ = stdin.Read(buf[1:2])
			_, _ = stdin.Read(buf[2:3])
			code := buf[2]
			switch code {
			case 'A':
				if selection > 0 {
					selection--
				} else {
					selection = len(repos) - 1
				}
			case 'B':
				if selection < len(repos)-1 {
					selection++
				} else {
					selection = 0
				}
			case 'C':
				if currentPage < totalPages-1 {
					currentPage++
					repos, err = fetchPage(currentPage)
					if err != nil {
						log.Fatalf("Error fetching repositories: %v", err)
					}
				}
			case 'D':
				if currentPage > 0 {
					currentPage--
					repos, err = fetchPage(currentPage)
					if err != nil {
						log.Fatalf("Error fetching repositories: %v", err)
					}
				}
			}
			continue
		}

		switch b {
		case 'q', 'Q':
			return
		case ' ':
			if selection >= 0 && selection < len(repos) {
				name := strings.ToLower(repos[selection].Name)
				if _, ok := watchedSet[name]; ok {
					delete(watchedSet, name)
				} else {
					watchedSet[name] = struct{}{}
				}
				saveWatched(watchedSet)
			}
		case '\r', '\n':
			if selection >= 0 && selection < len(repos) {
				name := strings.ToLower(repos[selection].Name)
				if _, ok := watchedSet[name]; ok {
					delete(watchedSet, name)
				} else {
					watchedSet[name] = struct{}{}
				}
				saveWatched(watchedSet)
			}
		case 'w', 'W':
			if len(watchedSet) > 0 {
				found := -1
				for i := selection + 1; i < len(repos); i++ {
					if _, ok := watchedSet[strings.ToLower(repos[i].Name)]; ok {
						found = i
						break
					}
				}
				if found == -1 {
					for i := 0; i <= selection && i < len(repos); i++ {
						if _, ok := watchedSet[strings.ToLower(repos[i].Name)]; ok {
							found = i
							break
						}
					}
				}
				if found != -1 {
					selection = found
				}
			}
		case 'g', 'G':
			fmt.Print("\nPage number: ")
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
			var pageInput string
			if _, err := fmt.Scanln(&pageInput); err != nil {
				pageInput = ""
			}
			oldState, _ = term.MakeRaw(int(os.Stdin.Fd()))

			if p, err := strconv.Atoi(strings.TrimSpace(pageInput)); err == nil && p >= 1 && p <= totalPages {
				currentPage = p - 1
				repos, err = fetchPage(currentPage)
				if err != nil {
					log.Fatalf("Error fetching repositories: %v", err)
				}
				selection = 0
			}
		case 's', 'S':
			fmt.Print("\nSaved. Press any key...")
			_, _ = stdin.Read(buf[:1])
		}
	}
}

func saveWatched(watchedSet map[string]struct{}) {
	cfg, _ := config.Load()
	var list []string
	for k := range watchedSet {
		list = append(list, k)
	}
	sort.Strings(list)
	cfg.Bitbucket.WatchedRepos = list
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save config: %v\n", err)
	}
}

func displayReposPage(repos []bitbucket.Repository, workspace string) {
	cfg, _ := config.Load()
	watched := map[string]struct{}{}
	for _, w := range cfg.Bitbucket.WatchedRepos {
		watched[strings.ToLower(w)] = struct{}{}
	}

	for idx, repo := range repos {
		privacyIcon := "🔓"
		if repo.IsPrivate {
			privacyIcon = "🔒"
		}
		watchIcon := "[ ]"
		if _, ok := watched[strings.ToLower(repo.Name)]; ok {
			watchIcon = "[⭐]"
		}
		fmt.Printf("%2d. %s %s %s 🔗 https://bitbucket.org/%s/%s", idx+1, watchIcon, privacyIcon, repo.Name, workspace, repo.Name)
		if repo.Language != "" {
			fmt.Printf(" 💻 %s", repo.Language)
		}
		fmt.Println()
	}
	fmt.Println("\nLegend: [⭐]=watched  [ ]=not watched | Use --interactive for arrow-key navigation")
}
