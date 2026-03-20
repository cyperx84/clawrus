# Command Reference

Complete reference for every clawrus command, flag, and option.

---

## Global Flags

These flags apply to all commands:

```
--model <string>        Override model for all threads in the run
--thinking <string>     Override thinking mode: off|low|medium|high
--timeout <int>         Per-thread timeout in seconds (default: 300)
--parallel <int>        Max concurrent thread operations (default: 4)
--gateway-url <url>     Override gateway URL (auto-discovered if omitted)
-v, --version           Print version and exit
-h, --help              Show help
```

---

## clawrus init

Create a sample `~/.clawrus/groups.yaml` with example structure.

```
Usage: clawrus init
```

Creates the `~/.clawrus/` directory if it doesn't exist and writes a starter config with placeholder thread IDs.

---

## clawrus run

Send a message to all threads in a group, preset, or ad-hoc list.

```
Usage: clawrus run [group|@preset] [message] [flags]
```

### Flags

```
--mode <string>              Run mode: broadcast (default), gather, pipeline, or poll
--gather-timeout <int>       Seconds to wait for replies in gather/pipeline/poll mode (default: 60)
--threads <string>           Comma-separated thread IDs for ad-hoc runs (no group needed)
```

### Examples

```bash
# Broadcast to a group
clawrus run backend "deploy to staging"

# Gather replies with LLM summary
clawrus run backend "status update" --mode gather --gather-timeout 120

# Pipeline: each thread's reply feeds the next
clawrus run chain "Research topic X" --mode pipeline --gather-timeout 90

# Poll: quick status sweep (table output, no LLM)
clawrus run @ops "What's your status?" --mode poll

# Run a preset
clawrus run @deploy "prepare release"

# Ad-hoc thread list
clawrus run --threads 111,222,333 "ping"

# Override model for the run
clawrus run backend "complex task" --model claude-opus-4-6 --thinking high

# Per-thread prompts (no message argument needed)
clawrus run my-sprint

# Template substitution in prompts
clawrus run agents "Hi {{name}}, update {{group}} on your progress for {{date}}"

# Limit concurrency
clawrus run backend "build" --parallel 2
```

### Run Modes

**Broadcast (default):**
Sends message to all threads in parallel. Prints a results table: thread, status (✅/❌), error if any.

**Gather:**
Sends message to all threads in parallel, then polls each for a reply (up to `--gather-timeout`).
Prints each reply, then runs an LLM summary via the OpenClaw gateway `/v1/chat/completions` endpoint.

**Pipeline:**
Runs threads sequentially. Sends the initial message to thread[0], waits for its reply, then passes that reply as the input to thread[1], and so on.
Use for multi-stage reasoning or refinement chains (e.g. research → draft → critique).
If a step times out, the previous reply (or initial message) is passed to the next thread.

**Poll:**
Sends message to all threads in parallel (same as broadcast), then collects replies.
Outputs a compact status table: THREAD | STATUS | REPLY (truncated to 80 chars).
No LLM summarization — fast lightweight fleet check.

### Template Substitution

Any message or per-thread prompt can use these placeholders:

| Placeholder | Replaced with |
|-------------|---------------|
| `{{name}}` | Thread name |
| `{{group}}` | Group name |
| `{{preset}}` | Preset name (empty if none) |
| `{{date}}` | Current date `YYYY-MM-DD` |
| `{{time}}` | Current time `HH:MM` (24h) |

Example:
```bash
clawrus group add sprint-1 123456 --name "BuildBot" \
  --prompt "Hey {{name}}, please summarize today ({{date}}) progress for group {{group}}"
```

### Priority Resolution

For model, thinking, and timeout:
```
CLI flag > per-thread override > group default > hardcoded default
```

For message:
```
Per-thread prompt > positional [message] argument
```

---

## clawrus list

List all groups with thread counts.

```
Usage: clawrus list
       clawrus ls
```

Alias for `clawrus group list`.

---

## clawrus show

Show detailed info for a group.

```
Usage: clawrus show <group>
```

Alias for `clawrus group show`.

---

## clawrus group new

Create an empty group.

```
Usage: clawrus group new <name>
```

---

## clawrus group delete

Delete a group and all its threads.

```
Usage: clawrus group delete <name>
```

---

## clawrus group add

Add a thread to a group.

```
Usage: clawrus group add <group> <thread-id> [flags]
```

### Flags

```
--name <string>       Human-readable label for the thread
--model <string>      Per-thread model override
--thinking <string>   Per-thread thinking mode: off|low|medium|high
--prompt <string>     Per-thread prompt (used instead of run message; supports {{placeholders}})
--context <string>    Additional context prepended to the prompt
```

### Examples

```bash
clawrus group add backend 1484056775278989333 --name "AuthAgent"
clawrus group add backend 1484056779322036234 --name "PayAgent" --model claude-opus-4-6
clawrus group add sprint-1 1484056767544688690 --name "SearchAgent" \
  --prompt "Hey {{name}}, migrate to Meilisearch. Report back by {{date}}."
```

---

## clawrus group remove

Remove a thread from a group by ID or name.

```
Usage: clawrus group remove <group> <thread-id-or-name>
```

### Examples

```bash
clawrus group remove backend 1484056775278989333
clawrus group remove backend AuthAgent
```

---

## clawrus group list

List all groups with thread counts, default model, and timeout.

```
Usage: clawrus group list
       clawrus group ls
```

---

## clawrus group show

Show all threads in a group with their IDs, names, models, and prompts.

```
Usage: clawrus group show <group>
```

---

## clawrus group clone

Deep-copy a group with all its threads and settings.

```
Usage: clawrus group clone <source> <destination>
```

---

## clawrus group set

Update group-level defaults.

```
Usage: clawrus group set <name> [flags]
```

### Flags

```
--model <string>      Default model for threads without a per-thread override
--thinking <string>   Default thinking mode: off|low|medium|high
--timeout <int>       Default timeout in seconds
```

### Examples

```bash
clawrus group set backend --model claude-opus-4-6 --thinking high --timeout 600
```

---

## clawrus group set-prompt

Set or update the per-thread prompt for a specific thread.

```
Usage: clawrus group set-prompt <group> <thread-id-or-name> <prompt>
```

### Examples

```bash
clawrus group set-prompt sprint-1 AuthAgent "Implement SSO with OpenID Connect"
clawrus group set-prompt sprint-1 1484056775278989333 "Build the auth flow"
```

---

## clawrus preset new

Create a named preset from one or more groups. Presets use the `@` prefix.

```
Usage: clawrus preset new <name> <group1> [group2...]
```

The `@` prefix is optional in the name — it's added automatically.

### Examples

```bash
clawrus preset new deploy backend frontend infra
clawrus preset new @daily-check monitoring alerts
```

---

## clawrus preset delete

Delete a preset.

```
Usage: clawrus preset delete <name>
```

---

## clawrus preset list

List all presets with their group memberships.

```
Usage: clawrus preset list
       clawrus preset ls
```

---

## clawrus preset show

Show all threads in a preset (expanded from its groups, deduplicated).

```
Usage: clawrus preset show <name>
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `CLAWRUS_CONFIG` | Override path to `groups.yaml` (default: `~/.clawrus/groups.yaml`) |
| `OPENCLAW_TOKEN` | Auth token for the OpenClaw gateway |
