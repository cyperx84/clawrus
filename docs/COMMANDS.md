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
--mode <string>              Run mode: broadcast (default) or gather
--gather-timeout <int>       Seconds to wait for replies in gather mode (default: 60)
--threads <string>           Comma-separated thread IDs for ad-hoc runs (no group needed)
```

### Examples

```bash
# Broadcast to a group
clawrus run backend "deploy to staging"

# Gather replies with timeout
clawrus run backend "status update" --mode gather --gather-timeout 120

# Run a preset
clawrus run @deploy "prepare release"

# Ad-hoc thread list
clawrus run --threads 111,222,333 "ping"

# Override model for the run
clawrus run backend "complex task" --model claude-opus-4-6 --thinking high

# Per-thread prompts (no message argument)
clawrus run my-sprint

# Limit concurrency
clawrus run backend "build" --parallel 2
```

### Behavior

**Broadcast mode (default):**
1. Sends message to each thread in parallel (bounded by `--parallel`)
2. Prints a results table: thread name/ID, status (OK/FAIL), error if any

**Gather mode:**
1. Sends message to each thread
2. Polls each thread for a reply every 3 seconds until `--gather-timeout`
3. Prints each thread's reply
4. Attempts LLM summarization via `/api/ai/complete` (gracefully skipped if unavailable)

**Per-thread prompts:** If a thread has a `prompt` field, that prompt is sent instead of the positional `[message]` argument. If all threads have prompts, the message argument can be omitted entirely.

**Priority resolution for model/thinking/timeout:**
`CLI flag > per-thread override > group default > hardcoded default`

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
--prompt <string>     Per-thread prompt (used instead of run message)
```

### Examples

```bash
clawrus group add backend 1484056775278989333 --name "AuthAgent"
clawrus group add backend 1484056779322036234 --name "PayAgent" --model claude-opus-4-6
clawrus group add sprint-1 1484056767544688690 --name "SearchAgent" --prompt "Migrate to Meilisearch"
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
