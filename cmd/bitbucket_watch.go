package cmd

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"devflow/internal/config"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Manage watched repositories",
	Long:  "Add, remove, list and toggle watched Bitbucket repositories maintained in local config.",
}

var watchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List watched repositories",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("load config: %v", err)
		}
		if len(cfg.Bitbucket.WatchedRepos) == 0 {
			fmt.Println("No watched repositories.")
			return
		}
		list := append([]string{}, cfg.Bitbucket.WatchedRepos...)
		sort.Strings(list)
		fmt.Println("Watched repositories:")
		for _, r := range list {
			fmt.Println(" -", r)
		}
	},
}

var watchAddCmd = &cobra.Command{
	Use:   "add <repo> [repo2 ...]",
	Short: "Add repositories to watched list",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		modifyWatched(args, nil)
	},
}

var watchRemoveCmd = &cobra.Command{
	Use:     "remove <repo> [repo2 ...]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove repositories from watched list",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		modifyWatched(nil, args)
	},
}

var watchToggleCmd = &cobra.Command{
	Use:   "toggle <repo> [repo2 ...]",
	Short: "Toggle repositories in watched list",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("load config: %v", err)
		}
		set := make(map[string]struct{}, len(cfg.Bitbucket.WatchedRepos))
		for _, r := range cfg.Bitbucket.WatchedRepos {
			set[strings.ToLower(r)] = struct{}{}
		}
		changed := false
		for _, a := range args {
			key := strings.ToLower(a)
			if _, ok := set[key]; ok {
				delete(set, key)
			} else {
				set[key] = struct{}{}
			}
			changed = true
		}
		if changed {
			cfg.Bitbucket.WatchedRepos = setToSortedSlice(set)
			if err := config.Save(cfg); err != nil {
				log.Fatalf("save config: %v", err)
			}
		}
		printWatchedSummary(cfg.Bitbucket.WatchedRepos)
	},
}

func modifyWatched(add []string, remove []string) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	set := make(map[string]struct{}, len(cfg.Bitbucket.WatchedRepos))
	for _, r := range cfg.Bitbucket.WatchedRepos {
		set[strings.ToLower(r)] = struct{}{}
	}

	for _, a := range add {
		set[strings.ToLower(a)] = struct{}{}
	}
	for _, r := range remove {
		delete(set, strings.ToLower(r))
	}

	cfg.Bitbucket.WatchedRepos = setToSortedSlice(set)
	if err := config.Save(cfg); err != nil {
		log.Fatalf("save config: %v", err)
	}
	printWatchedSummary(cfg.Bitbucket.WatchedRepos)
}

func setToSortedSlice(set map[string]struct{}) []string {
	list := make([]string, 0, len(set))
	for k := range set {
		list = append(list, k)
	}
	sort.Strings(list)
	return list
}

func printWatchedSummary(list []string) {
	if len(list) == 0 {
		fmt.Println("No watched repositories.")
		return
	}
	fmt.Printf("Watched (%d): %s\n", len(list), strings.Join(list, ", "))
}

func init() {
	watchCmd.AddCommand(watchListCmd)
	watchCmd.AddCommand(watchAddCmd)
	watchCmd.AddCommand(watchRemoveCmd)
	watchCmd.AddCommand(watchToggleCmd)
	repoCmd.AddCommand(watchCmd)
}
