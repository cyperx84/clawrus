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
	flagModel      string
	flagThinking   string
	flagTimeout    int
	flagParallel   int
	flagGatewayURL string
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
	root.PersistentFlags().StringVar(&flagGatewayURL, "gateway-url", "", "OpenClaw gateway URL (auto-discovered if not set)")

	root.AddCommand(groupCmd())
	root.AddCommand(runCmd())
	root.AddCommand(initCmd())

	// Top-level aliases for group list and group show
	root.AddCommand(listAliasCmd())
	root.AddCommand(showAliasCmd())

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
	mainCfg, _ := config.LoadMainConfig()
	configURL := ""
	if mainCfg != nil {
		configURL = mainCfg.Gateway.URL
	}

	baseURL, err := gateway.DiscoverGateway(flagGatewayURL, configURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	return gateway.NewClient(baseURL, "", "")
}

// groupCmd is the parent command for all group management subcommands
func groupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Manage thread groups",
		Long:  "Create, modify, and inspect thread groups. All changes are persisted to ~/.clawrus/groups.yaml.",
	}
	cmd.AddCommand(groupNewCmd())
	cmd.AddCommand(groupDeleteCmd())
	cmd.AddCommand(groupAddCmd())
	cmd.AddCommand(groupRemoveCmd())
	cmd.AddCommand(groupListCmd())
	cmd.AddCommand(groupShowCmd())
	cmd.AddCommand(groupCloneCmd())
	cmd.AddCommand(groupSetCmd())
	cmd.AddCommand(groupSetPromptCmd())
	return cmd
}

// groupNewCmd creates an empty group
func groupNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <name>",
		Short: "Create an empty group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			if _, exists := cfg.Groups[name]; exists {
				return fmt.Errorf("group %q already exists", name)
			}
			cfg.Groups[name] = types.Group{Threads: []types.Thread{}}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Created group %q\n", name)
			return nil
		},
	}
}

// groupDeleteCmd dissolves a group
func groupDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			if _, exists := cfg.Groups[name]; !exists {
				return fmt.Errorf("group %q not found", name)
			}
			delete(cfg.Groups, name)
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Deleted group %q\n", name)
			return nil
		},
	}
}

// groupAddCmd adds a thread to a group
func groupAddCmd() *cobra.Command {
	var (
		addName     string
		addModel    string
		addThinking string
		addPrompt   string
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
				return fmt.Errorf("group %q not found (use `clawrus group new %s` first)", groupName, groupName)
			}
			for _, t := range g.Threads {
				if t.ID == threadID {
					return fmt.Errorf("thread %s already in group %s", threadID, groupName)
				}
			}
			label := addName
			if label == "" {
				label = threadID
			}
			thread := types.Thread{ID: threadID, Name: label}
			if addModel != "" {
				thread.Model = addModel
			}
			if addThinking != "" {
				thread.Thinking = addThinking
			}
			if addPrompt != "" {
				thread.Prompt = addPrompt
			}
			g.Threads = append(g.Threads, thread)
			cfg.Groups[groupName] = g
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Added %s to %s\n", label, groupName)
			return nil
		},
	}
	cmd.Flags().StringVar(&addName, "name", "", "Thread label (defaults to thread ID)")
	cmd.Flags().StringVar(&addModel, "model", "", "Thread-specific model override")
	cmd.Flags().StringVar(&addThinking, "thinking", "", "Thread-specific thinking override")
	cmd.Flags().StringVar(&addPrompt, "prompt", "", "Per-thread prompt (sent instead of positional message)")
	return cmd
}

// groupRemoveCmd removes a thread by ID or name label
func groupRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <group> <thread-id-or-label>",
		Short: "Remove a thread from a group (by ID or name)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			groupName, needle := args[0], args[1]
			g, ok := cfg.Groups[groupName]
			if !ok {
				return fmt.Errorf("group %q not found", groupName)
			}
			found := false
			newThreads := []types.Thread{}
			for _, t := range g.Threads {
				if t.ID == needle || t.Name == needle {
					found = true
				} else {
					newThreads = append(newThreads, t)
				}
			}
			if !found {
				return fmt.Errorf("thread %q not found in group %s (checked both ID and name)", needle, groupName)
			}
			g.Threads = newThreads
			cfg.Groups[groupName] = g
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Removed %s from %s\n", needle, groupName)
			return nil
		},
	}
}

// groupCloneCmd deep-copies a group
func groupCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <src> <dst>",
		Short: "Clone a group into a new name",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			src, dst := args[0], args[1]
			g, ok := cfg.Groups[src]
			if !ok {
				return fmt.Errorf("source group %q not found", src)
			}
			if _, exists := cfg.Groups[dst]; exists {
				return fmt.Errorf("destination group %q already exists", dst)
			}
			// Deep copy threads
			cloned := types.Group{
				Model:    g.Model,
				Thinking: g.Thinking,
				Timeout:  g.Timeout,
				Threads:  make([]types.Thread, len(g.Threads)),
			}
			copy(cloned.Threads, g.Threads)
			cfg.Groups[dst] = cloned
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Cloned %s → %s (%d threads)\n", src, dst, len(cloned.Threads))
			return nil
		},
	}
}

// groupSetCmd updates group-level defaults without touching threads
func groupSetCmd() *cobra.Command {
	var (
		setModel    string
		setThinking string
		setTimeout  int
	)
	cmd := &cobra.Command{
		Use:   "set <name>",
		Short: "Update group defaults (model, thinking, timeout)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			g, ok := cfg.Groups[name]
			if !ok {
				return fmt.Errorf("group %q not found", name)
			}
			changed := false
			if cmd.Flags().Changed("model") {
				g.Model = setModel
				changed = true
			}
			if cmd.Flags().Changed("thinking") {
				g.Thinking = setThinking
				changed = true
			}
			if cmd.Flags().Changed("timeout") {
				g.Timeout = setTimeout
				changed = true
			}
			if !changed {
				return fmt.Errorf("no flags provided; use --model, --thinking, or --timeout")
			}
			cfg.Groups[name] = g
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Updated group %q\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&setModel, "model", "", "Group default model")
	cmd.Flags().StringVar(&setThinking, "thinking", "", "Group default thinking mode")
	cmd.Flags().IntVar(&setTimeout, "timeout", 0, "Group default timeout in seconds")
	return cmd
}

// groupSetPromptCmd sets/updates the prompt for an existing thread in a group
func groupSetPromptCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-prompt <group> <thread-id-or-label> <prompt>",
		Short: "Set or update the prompt for a thread in a group",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			groupName, needle, prompt := args[0], args[1], args[2]
			g, ok := cfg.Groups[groupName]
			if !ok {
				return fmt.Errorf("group %q not found", groupName)
			}
			found := false
			for i, t := range g.Threads {
				if t.ID == needle || t.Name == needle {
					g.Threads[i].Prompt = prompt
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("thread %q not found in group %s (checked both ID and name)", needle, groupName)
			}
			cfg.Groups[groupName] = g
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Set prompt for %s in %s\n", needle, groupName)
			return nil
		},
	}
}

// groupListCmd shows all groups
func groupListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all thread groups",
		Aliases: []string{"ls"},
		RunE:    listRunE,
	}
}

// groupShowCmd shows details of a single group
func groupShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <group>",
		Short: "Show group details and thread list",
		Args:  cobra.ExactArgs(1),
		RunE:  showRunE,
	}
}

// listAliasCmd is a top-level shortcut for `group list`
func listAliasCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all thread groups (alias for group list)",
		Aliases: []string{"ls"},
		RunE:    listRunE,
	}
}

// showAliasCmd is a top-level shortcut for `group show`
func showAliasCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <group>",
		Short: "Show group details (alias for group show)",
		Args:  cobra.ExactArgs(1),
		RunE:  showRunE,
	}
}

// listRunE is the shared implementation for list commands
func listRunE(cmd *cobra.Command, args []string) error {
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
}

// showRunE is the shared implementation for show commands
func showRunE(cmd *cobra.Command, args []string) error {
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
	fmt.Fprintln(w, "  NAME\tID\tMODEL\tTHINKING\tPROMPT")
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
		prompt := t.Prompt
		if prompt == "" {
			prompt = "-"
		} else if len(prompt) > 40 {
			prompt = prompt[:37] + "..."
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n", name, t.ID, model, thinking, prompt)
	}
	w.Flush()
	return nil
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
			fmt.Println("\nNo configuration needed — clawrus auto-discovers the local OpenClaw gateway.")
			fmt.Println("Just edit the thread IDs above and run: clawrus run example \"hello\"")
			return nil
		},
	}
}

// runCmd fans out a command to threads — either from a named group or ad-hoc --threads
func runCmd() *cobra.Command {
	var (
		flagMode          string
		flagGatherTimeout int
		flagThreads       string
	)
	cmd := &cobra.Command{
		Use:   "run [<group>] <command...>",
		Short: "Send a command to all threads in a group or ad-hoc thread list",
		Long: `Send a command to threads. Two modes:

  Named group:   clawrus run <group-name> "message"
  Ad-hoc:        clawrus run --threads <id1>,<id2> "message"

Ad-hoc runs are ephemeral — nothing is saved to config.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var threads []types.Thread
			var groupName string
			var g types.Group

			if flagThreads != "" {
				// Ad-hoc mode: --threads flag provided
				ids := strings.Split(flagThreads, ",")
				for _, id := range ids {
					id = strings.TrimSpace(id)
					if id == "" {
						continue
					}
					threads = append(threads, types.Thread{ID: id, Name: id})
				}
				if len(threads) == 0 {
					return fmt.Errorf("--threads requires at least one thread ID")
				}
				g = types.Group{Threads: threads}
				groupName = "(ad-hoc)"
				// All args are the command
				if len(args) == 0 {
					return fmt.Errorf("command required")
				}
			} else {
				// Named group mode
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				groupName = args[0]
				args = args[1:]
				var ok bool
				g, ok = cfg.Groups[groupName]
				if !ok {
					return fmt.Errorf("group %q not found", groupName)
				}
				threads = g.Threads

				// Check if any thread has a per-thread prompt
				hasPerThreadPrompts := false
				for _, t := range threads {
					if t.Prompt != "" {
						hasPerThreadPrompts = true
						break
					}
				}
				// Require a command arg unless threads have per-thread prompts
				if len(args) == 0 && !hasPerThreadPrompts {
					return fmt.Errorf("usage: clawrus run <group> <command...> or clawrus run --threads <ids> <command...>")
				}
			}

			command := strings.Join(args, " ")

			gw := getGateway()
			results := make([]types.RunResult, len(threads))
			var wg sync.WaitGroup
			sem := make(chan struct{}, flagParallel)

			for i, t := range threads {
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

					// Per-thread prompt takes priority over positional command
					msg := command
					if thread.Prompt != "" {
						msg = thread.Prompt
					}

					fmt.Printf("→ %s: %q\n", name, msg)

					resp, err := gw.SendMessage(thread.ID, msg, model, thinking, timeout)
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
	cmd.Flags().StringVar(&flagThreads, "threads", "", "Comma-separated thread IDs for ad-hoc run (no group needed)")
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

	// Summarize via OpenClaw gateway
	fmt.Println()
	summary, err := gateway.SummarizeReplies(gw.BaseURL, repliesText.String())
	if err != nil {
		fmt.Printf("📋 Summary: (LLM error: %s)\n", err)
	} else if summary == "" {
		// /api/ai/complete not available — print raw replies
		fmt.Printf("📋 Raw replies:\n%s", repliesText.String())
	} else {
		fmt.Printf("📋 Summary: %s\n", summary)
	}

	return nil
}
