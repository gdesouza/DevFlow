package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// gitRepoStatus holds the computed sync information for a repository
type gitRepoStatus struct {
	Path     string `json:"path"`
	Branch   string `json:"branch"`
	State    string `json:"state"`
	Dirty    bool   `json:"dirty"`
	Stashed  bool   `json:"stashed"`
	Ahead    int    `json:"ahead"`
	Behind   int    `json:"behind"`
	Upstream string `json:"upstream"`
}

var (
	gitListPath    string
	gitListNoFetch bool
	gitListJSON    bool
	gitListTabular bool
)

var gitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List local git repositories and their sync status",
	Long: `Recursively discover git repositories under a path (current directory by default)
 and show their branch, sync state (up-to-date, ahead, behind, diverged, no-upstream, detached),
 cleanliness (dirty/clean), ahead/behind counts, and upstream branch.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		root := gitListPath
		if root == "" {
			root = "."
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}
		info, err := os.Stat(absRoot)
		if err != nil {
			return fmt.Errorf("stat path: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("path is not a directory: %s", absRoot)
		}

		repos, err := discoverGitRepos(absRoot)
		if err != nil {
			return err
		}
		// Concurrency: evaluate repositories in parallel with bounded workers
		workerCount := runtime.NumCPU()
		if workerCount < 2 {
			workerCount = 2
		}
		jobs := make(chan string)
		results := make(chan *gitRepoStatus, len(repos))
		var wg sync.WaitGroup

		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for repo := range jobs {
					st := evaluateRepo(absRoot, repo, !gitListNoFetch)
					if st != nil {
						results <- st
					}
				}
			}()
		}
		for _, r := range repos {
			jobs <- r
		}
		close(jobs)

		// Streaming mode (default, when not tabular or JSON): print each repo as soon as processed
		if !gitListTabular && !gitListJSON {
			// Collect results concurrently via another goroutine
			go func() {
				wg.Wait()
				close(results)
			}()
			stream := make([]gitRepoStatus, 0, len(repos))
			for r := range results {
				stream = append(stream, *r)
				fmt.Printf("%s\t%s\t%s\n", r.Path, r.Branch, r.State)
			}
			return nil
		}

		wg.Wait()
		close(results)
		slices := make([]gitRepoStatus, 0, len(repos))
		for r := range results {
			slices = append(slices, *r)
		}
		sort.Slice(slices, func(i, j int) bool { return slices[i].Path < slices[j].Path })

		if gitListJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(slices)
		}

		printRepoStatuses(slices)
		_ = start // reserved for potential timing output
		return nil
	},
}

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Local git utilities",
	Long:  "Utilities for interacting with local git repositories (discovery, status, etc.)",
}

func init() {
	gitCmd.AddCommand(gitListCmd)
	rootCmd.AddCommand(gitCmd)

	gitListCmd.Flags().StringVarP(&gitListPath, "path", "p", ".", "Root path to search recursively for git repositories")
	gitListCmd.Flags().BoolVar(&gitListNoFetch, "no-fetch", false, "Do not run 'git fetch' (faster, but may show stale upstream info)")
	gitListCmd.Flags().BoolVar(&gitListJSON, "json", false, "Output JSON list")
	gitListCmd.Flags().BoolVar(&gitListTabular, "tabular", false, "Render full table (waits for all repos)")
}

func discoverGitRepos(root string) ([]string, error) {
	repos := []string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			repo := filepath.Dir(path)
			repos = append(repos, repo)
			return filepath.SkipDir // don't descend into .git
		}
		return nil
	})
	return repos, err
}

func evaluateRepo(root, repoPath string, doFetch bool) *gitRepoStatus {
	// Open repository (worktree)
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil
	}
	wt, _ := r.Worktree()
	branchRef, _ := r.Head()
	branchName := "DETACHED"
	if branchRef != nil && branchRef.Name().IsBranch() {
		branchName = branchRef.Name().Short()
	}

	st := gitRepoStatus{Path: relativePath(root, repoPath), Branch: branchName}

	// Determine upstream (tracking) reference
	upstream := ""
	if branchRef != nil && branchRef.Name().IsBranch() {
		cfg, _ := r.Config()
		for name, b := range cfg.Branches {
			if name == branchName && b.Remote != "" && b.Merge != "" {
				upstream = fmt.Sprintf("%s/%s", b.Remote, b.Merge.Short())
				break
			}
		}
	}
	st.Upstream = upstream

	// Fetch (optionally) for each remote referenced by the branch
	if doFetch && upstream != "" {
		parts := strings.SplitN(upstream, "/", 2)
		remoteName := parts[0]
		rem, err := r.Remote(remoteName)
		if err == nil {
			_ = rem.Fetch(&git.FetchOptions{Tags: git.AllTags, Force: false})
		}
	}

	// Dirty status
	if wt != nil {
		if status, err := wt.Status(); err == nil {
			st.Dirty = !status.IsClean()
		}
	}
	// Stash detection: presence of refs/stash file or stash reflog entries
	gitDir := filepath.Join(repoPath, ".git")
	if fi, err := os.Stat(filepath.Join(gitDir, "logs", "refs", "stash")); err == nil && !fi.IsDir() {
		st.Stashed = true
	} else if fi, err := os.Stat(filepath.Join(gitDir, "refs", "stash")); err == nil && !fi.IsDir() {
		st.Stashed = true
	}

	// Ahead/Behind computation
	if upstream == "" && branchName == "DETACHED" {
		st.State = "detached"
		return &st
	} else if upstream == "" {
		st.State = "no-upstream"
		return &st
	}

	// Resolve local and remote commit hashes
	localHash := branchRef.Hash()
	remoteParts := strings.SplitN(upstream, "/", 2)
	remoteRefName := plumbing.NewRemoteReferenceName(remoteParts[0], remoteParts[1])
	remoteRef, err := r.Reference(remoteRefName, true)
	if err != nil {
		// remote ref missing
		st.State = "no-upstream"
		return &st
	}
	remoteHash := remoteRef.Hash()

	// Build ancestor sets (bounded) for ahead/behind approximation
	maxCommits := 2000 // safety bound
	localAnc := map[plumbing.Hash]struct{}{}
	remoteAnc := map[plumbing.Hash]struct{}{}
	queue := []plumbing.Hash{localHash}
	for len(queue) > 0 && len(localAnc) < maxCommits {
		h := queue[0]
		queue = queue[1:]
		if _, seen := localAnc[h]; seen {
			continue
		}
		localAnc[h] = struct{}{}
		c, err := r.CommitObject(h)
		if err != nil {
			continue
		}
		queue = append(queue, c.ParentHashes...)
	}
	queue = []plumbing.Hash{remoteHash}
	for len(queue) > 0 && len(remoteAnc) < maxCommits {
		h := queue[0]
		queue = queue[1:]
		if _, seen := remoteAnc[h]; seen {
			continue
		}
		remoteAnc[h] = struct{}{}
		c, err := r.CommitObject(h)
		if err != nil {
			continue
		}
		queue = append(queue, c.ParentHashes...)
	}
	// Count ahead/behind (exclude common ancestors)
	ahead := 0
	for h := range localAnc {
		if _, ok := remoteAnc[h]; !ok {
			ahead++
		}
	}
	behind := 0
	for h := range remoteAnc {
		if _, ok := localAnc[h]; !ok {
			behind++
		}
	}
	// Do not count the head commit twice
	if _, ok := remoteAnc[localHash]; ok {
		ahead--
	}
	if _, ok := localAnc[remoteHash]; ok {
		behind--
	}
	if ahead < 0 {
		ahead = 0
	}
	if behind < 0 {
		behind = 0
	}
	st.Ahead = ahead
	st.Behind = behind

	if localHash == remoteHash {
		st.State = "up-to-date"
	} else if ahead > 0 && behind == 0 {
		st.State = "ahead"
	} else if behind > 0 && ahead == 0 {
		st.State = "behind"
	} else {
		st.State = "diverged"
	}
	return &st
}

// atoiSafe parses integer from string returning 0 on error
func atoiSafe(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func relativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return "."
	}
	return rel
}

func colorize(cond bool, color string, text string) string {
	if !cond {
		return text
	}
	return color + text + "\033[0m"
}

func printRepoStatuses(repos []gitRepoStatus) {
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	// Use go-pretty for clean formatting
	importedTable := false
	_ = importedTable
	// Lazy import usage pattern: build rows then render
	// Coloring applied per-cell when TTY
	stateColor := func(state string) string {
		if !isTTY {
			return state
		}
		switch state {
		case "up-to-date":
			return colorize(true, "\033[32m", state)
		case "ahead":
			return colorize(true, "\033[33m", state)
		case "behind":
			return colorize(true, "\033[31m", state)
		case "diverged":
			return colorize(true, "\033[35m", state)
		case "no-upstream", "detached":
			return colorize(true, "\033[36m", state)
		}
		return state
	}
	dirtyColor := func(d bool) string {
		if d {
			return "dirty"
		}
		return "clean"
	}
	if isTTY {
		dirtyColor = func(d bool) string {
			if d {
				return colorize(true, "\033[31m", "dirty")
			}
			return colorize(true, "\033[32m", "clean")
		}
	}
	// Build table using go-pretty
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Repository", "Branch", "State", "Dirty", "Stashed", "Ahead", "Behind"})
	for _, r := range repos {
		stashedVal := "no"
		if r.Stashed {
			stashedVal = "yes"
		}
		row := table.Row{truncate(r.Path, 55), truncate(r.Branch, 40), stateColor(r.State), dirtyColor(r.Dirty), stashedVal, r.Ahead, r.Behind}
		t.AppendRow(row)
	}
	t.SetStyle(table.StyleRounded)
	t.Render()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
