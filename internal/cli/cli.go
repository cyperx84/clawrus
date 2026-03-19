package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/cyperx84/clawrus/internal/config"
	"github.com/cyperx84/clawrus/internal/gateway"
	"github.com/cyperx84/clawrus/internal/types"
	"github.com/spf13/cobra"
)

var (
	flagModel    string
	flagThinking string
	flagTimeout  int
	flagParallel int
)

func RootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "clawrus",
		Short: "Clawrus — agent thread orchestration for OpenClaw",
		Long:  "Manage and run commands against groups of OpenClaw Discord threads.",
	}
	root.PersistentFlags().StringVar(&flagModel, "model", "", "Override model for all threads")
	root.PersistentFlags().StringVar(&flagThinking, "thinking", "", "Override thinking mode (off|low|medium|high)")
	root.PersistentFlags().IntVar(&flagTimeout, "timeout", 300, "Per-thread timeout in seconds")
	root.PersistentFlags().IntVar(&flagParallel, "parallel", 4, "Max concurrent threads")

	root.AddCommand(listCmd())
	root.AddCommand(runCmd())
	root.AddCommand(addCmd())
	root.AddCommand(removeCmd())
	root.AddCommand(showCmd())
	root.AddCommand(initCmd())

	return root
}

func resolveModel(groupModel, threadModel, flagModel string) string {
	if flagModel != "" {
		return flagModel
	}
	if threadModel != "" {
		return threadModel
	}
	return groupModel
}

func resolveThinking(groupThinking, threadThinking, flagThinking string) string {
	if flagThinking != "" {
		return flagThinking
	}
	if threadThinking != "" {
		return threadThinking
	}
	return groupThinking
}

func resolveTimeout(groupTimeout, threadTimeout, flagTimeout int) time.Duration {
	if flagTimeout != 0 {
		return time.Duration(flagTimeout) * time.Second
	}
	if threadTimeout != 0 {
		return time.Duration(threadTimeout) * time.Second
	}
	if groupTimeout != 0 {
		return time.Duration(groupTimeout) * time.Second
	}
	return 300 * time.Second
}

func getGateway() *gateway.Client {
	baseURL := os.Getenv("OPENCLAW_URL")
	apiKey := os.Getenv("OPENCLAW_API_KEY")
	agentID := os.Getenv("OPENCLAW_AGENT_ID")
	if baseURL == "" {
		baseURL = "http://localhost:3260"
	}
	return gateway.NewClient(baseURL, apiKey, agentID)
}

// initCmd creates a sample config
func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create sample groups.yaml config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := &types.GroupConfig{
				Groups: map[string]types.Group{
					"example": {
						Model:   "glm-5-turbo",
						Timeout: 300,
						Threads: []types.Thread{
							{ID: "CHANNEL_ID_HERE", Name: "Thread 1"},
							{ID: "CHANNEL_ID_HERE", Name: "Thread 2"},
						},
					},
				},
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Created %s\n", config.ConfigPath())
			return nil
		},
	}
}

// listCmd shows all groups
func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all thread groups",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Groups) == 0 {
				fmt.Println("No groups configured. Run `clawrus init` to create a sample config.")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "GROUP\tTHREADS\tMODEL\tTHINKING\tTIMEOUT")
			for name, g := range cfg.Groups {
				model := g.Model
				if flagModel != "" {
					model = flagModel + " (override)"
				}
				thinking := g.Thinking
				if flagThinking != "" {
					thinking = flagThinking + " (override)"
				}
				if thinking == "" {
					thinking = "-"
				}
				if model == "" {
					model = "(default)"
				}
				timeout := fmt.Sprintf("%ds", g.Timeout)
				if g.Timeout == 0 {
					timeout = "300s"
				}
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", name, len(g.Threads), model, thinking, timeout)
			}
			w.Flush()
			return nil
		},
	}
}

// showCmd shows details of a single group
func showCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <group>",
		Short: "Show group details and thread list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			g, ok := cfg.Groups[args[0]]
			if !ok {
				return fmt.Errorf("group %q not found", args[0])
			}
			fmt.Printf("Group: %s\n", args[0])
			if g.Model != "" {
				fmt.Printf("  Model:    %s\n", g.Model)
			}
			if g.Thinking != "" {
				fmt.Printf("  Thinking: %s\n", g.Thinking)
			}
			if g.Timeout != 0 {
				fmt.Printf("  Timeout:  %ds\n", g.Timeout)
			}
			fmt.Printf("  Threads:  %d\n\n", len(g.Threads))
			w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "  NAME\tID\tMODEL\tTHINKING")
			for _, t := range g.Threads {
				name := t.Name
				if name == "" {
					name = "(unnamed)"
				}
				model := t.Model
				if model == "" {
					model = "(inherit)"
				}
				thinking := t.Thinking
				if thinking == "" {
					thinking = "(inherit)"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", name, t.ID, model, thinking)
			}
			w.Flush()
			return nil
		},
	}
}

// runCmd fans out a command to all threads in a group
func runCmd() *cobra.Command {
	var (
		flagMode          string
		flagGatherTimeout int
	)
	cmd := &cobra.Command{
		Use:   "run <group> <command...>",
		Short: "Send a command to all threads in a group",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			groupName := args[0]
			command := strings.Join(args[1:], " ")

			g, ok := cfg.Groups[groupName]
			if !ok {
				return fmt.Errorf("group %q not found", args[0])
			}

			gw := getGateway()
			results := make([]types.RunResult, len(g.Threads))
			var wg sync.WaitGroup
			sem := make(chan struct{}, flagParallel)

			for i, t := range g.Threads {
				wg.Add(1)
				go func(idx int, thread types.Thread) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					model := resolveModel(g.Model, thread.Model, flagModel)
					thinking := resolveThinking(g.Thinking, thread.Thinking, flagThinking)
					timeout := resolveTimeout(g.Timeout, thread.Timeout, flagTimeout)

					name := thread.Name
					if name == "" {
						name = thread.ID
					}

					resp, err := gw.SendMessage(thread.ID, command, model, thinking, timeout)
					if err != nil {
						results[idx] = types.RunResult{ThreadID: thread.ID, ThreadName: name, OK: false, Error: err.Error()}
						return
					}
					results[idx] = types.RunResult{ThreadID: thread.ID, ThreadName: name, OK: resp.OK, Error: resp.Error}
				}(i, t)
			}
			wg.Wait()

			if flagMode == "gather" {
				return runGather(gw, g, groupName, results, time.Duration(flagGatherTimeout)*time.Second)
			}

			// Broadcast mode: print results table
			okCount := 0
			failCount := 0
			w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "THREAD\tSTATUS\tERROR")
			for _, r := range results {
				status := "✅"
				if !r.OK {
					status = "❌"
					failCount++
				} else {
					okCount++
				}
				errMsg := "-"
				if r.Error != "" {
					errMsg = r.Error
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", r.ThreadName, status, errMsg)
			}
			w.Flush()
			fmt.Printf("\n%d/%d succeeded\n", okCount, okCount+failCount)
			return nil
		},
	}
	cmd.Flags().StringVar(&flagMode, "mode", "broadcast", "Run mode: broadcast or gather")
	cmd.Flags().IntVar(&flagGatherTimeout, "gather-timeout", 60, "Gather mode timeout in seconds")
	return cmd
}

// runGather polls for replies from each thread and optionally summarizes them via LLM.
func runGather(gw *gateway.Client, g types.Group, groupName string, results []types.RunResult, gatherTimeout time.Duration) error {
	// Poll for replies in parallel
	var wg sync.WaitGroup
	for i, r := range results {
		if !r.OK {
			continue
		}
		wg.Add(1)
		go func(idx int, threadID string) {
			defer wg.Done()
			reply, err := gw.PollReply(threadID, "", gatherTimeout)
			if err != nil {
				results[idx].Reply = fmt.Sprintf("(poll error: %s)", err)
			} else if reply == "" {
				results[idx].Reply = "(no reply within timeout)"
			} else {
				results[idx].Reply = reply
			}
		}(i, r.ThreadID)
	}
	wg.Wait()

	// Build output
	fmt.Printf("\n🎵 Clawrus — Gather Results\n")
	fmt.Printf("Group: %s | Mode: gather | Threads: %d\n\n", groupName, len(g.Threads))

	var repliesText strings.Builder
	for _, r := range results {
		reply := r.Reply
		if reply == "" && !r.OK {
			reply = fmt.Sprintf("(send failed: %s)", r.Error)
		}
		fmt.Printf("[%s] %q\n", r.ThreadName, reply)
		if r.OK && r.Reply != "" {
			repliesText.WriteString(fmt.Sprintf("[%s]: %s\n", r.ThreadName, r.Reply))
		}
	}

	// Summarize
	fmt.Println()
	summary, err := gateway.SummarizeReplies(repliesText.String())
	if err != nil {
		fmt.Printf("📋 Summary: (LLM error: %s)\n", err)
	} else if summary == "" {
		// No API key — print raw
		fmt.Printf("📋 Summary: %s\n", repliesText.String())
	} else {
		fmt.Printf("📋 Summary: %s\n", summary)
	}

	return nil
}

// addCmd adds a thread to a group
func addCmd() *cobra.Command {
	var (
		addName     string
		addModel    string
		addThinking string
	)
	cmd := &cobra.Command{
		Use:   "add <group> <thread-id>",
		Short: "Add a thread to a group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			groupName, threadID := args[0], args[1]
			g, ok := cfg.Groups[groupName]
			if !ok {
				g = types.Group{Threads: []types.Thread{}}
				cfg.Groups[groupName] = g
			}
			// Check for duplicate
			for _, t := range g.Threads {
				if t.ID == threadID {
					return fmt.Errorf("thread %s already in group %s", threadID, groupName)
				}
			}
			thread := types.Thread{ID: threadID, Name: addName}
			if addModel != "" {
				thread.Model = addModel
			}
			if addThinking != "" {
				thread.Thinking = addThinking
			}
			g.Threads = append(g.Threads, thread)
			cfg.Groups[groupName] = g
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Added %s to %s\n", addName, groupName)
			return nil
		},
	}
	cmd.Flags().StringVar(&addName, "name", "", "Thread name")
	cmd.Flags().StringVar(&addModel, "model", "", "Thread-specific model override")
	cmd.Flags().StringVar(&addThinking, "thinking", "", "Thread-specific thinking override")
	return cmd
}

// removeCmd removes a thread from a group
func removeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <group> <thread-id>",
		Short: "Remove a thread from a group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			groupName, threadID := args[0], args[1]
			g, ok := cfg.Groups[groupName]
			if !ok {
				return fmt.Errorf("group %q not found", groupName)
			}
			found := false
			newThreads := []types.Thread{}
			for _, t := range g.Threads {
				if t.ID == threadID {
					found = true
				} else {
					newThreads = append(newThreads, t)
				}
			}
			if !found {
				return fmt.Errorf("thread %s not in group %s", threadID, groupName)
			}
			g.Threads = newThreads
			cfg.Groups[groupName] = g
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Removed %s from %s\n", threadID, groupName)
			return nil
		},
	}
}
