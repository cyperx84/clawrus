# Clawrus

**Orchestrate your AI agent fleet from the command line.**

Clawrus is a CLI tool that broadcasts commands to multiple OpenClaw agent threads simultaneously, collects their replies, and summarizes the results. Think of it as `xargs` for AI agents — define groups, fire off instructions, and gather status from your entire fleet in seconds.

## Install

```bash
# Homebrew
brew install cyperx84/tap/clawrus

# Go
go install github.com/cyperx84/clawrus@latest
```

## Quick Start

```bash
clawrus init                                    # scaffold ~/.clawrus/groups.yaml
vim ~/.clawrus/groups.yaml                      # add your thread IDs
clawrus run my-agents "what's your status?"     # broadcast to all threads
clawrus run my-agents "status?" --mode gather   # collect and summarize replies
clawrus run @all "ship it"                      # hit every agent via preset
```

No configuration needed beyond thread IDs — clawrus auto-discovers the OpenClaw gateway and auth token.

## Commands

| Command | Description |
|---------|-------------|
| `clawrus init` | Create sample `~/.clawrus/groups.yaml` |
| `clawrus run <group\|@preset> <message>` | Send message to all threads in a group or preset |
| `clawrus list` | List all groups |
| `clawrus show <group>` | Show group details with all threads |
| `clawrus group new <name>` | Create an empty group |
| `clawrus group delete <name>` | Delete a group |
| `clawrus group add <group> <thread-id>` | Add a thread (`--name`, `--model`, `--thinking`, `--prompt`) |
| `clawrus group remove <group> <id-or-name>` | Remove a thread by ID or label |
| `clawrus group list` | List all groups (same as `clawrus list`) |
| `clawrus group show <group>` | Show group details (same as `clawrus show`) |
| `clawrus group clone <src> <dst>` | Deep-copy a group |
| `clawrus group set <name>` | Update group defaults (`--model`, `--thinking`, `--timeout`) |
| `clawrus group set-prompt <group> <id-or-name> <prompt>` | Set per-thread prompt |
| `clawrus preset new <name> <group1> [group2...]` | Create a named preset from groups |
| `clawrus preset delete <name>` | Delete a preset |
| `clawrus preset list` | List all presets |
| `clawrus preset show <name>` | Show preset details with all threads |

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--model <string>` | — | Override model for all threads |
| `--thinking <string>` | — | Override thinking mode: `off\|low\|medium\|high` |
| `--timeout <seconds>` | `300` | Per-thread timeout |
| `--parallel <int>` | `4` | Max concurrent threads |
| `--gateway-url <url>` | auto | Override gateway URL |
| `--mode <string>` | `broadcast` | Run mode: `broadcast\|gather` |
| `--gather-timeout <seconds>` | `60` | How long to wait for replies in gather mode |
| `--threads <ids>` | — | Ad-hoc comma-separated thread IDs (skip group) |

## Groups

Groups are named collections of agent threads stored in `~/.clawrus/groups.yaml`.

```bash
# Create and populate a group
clawrus group new backend-agents
clawrus group add backend-agents 1484056775278989333 --name "AuthAgent" --model claude-sonnet-4-6
clawrus group add backend-agents 1484056779322036234 --name "PaymentAgent"
clawrus group add backend-agents 1484056767544688690 --name "SearchAgent"

# Run against the group
clawrus run backend-agents "run your test suites and report results"

# Clone for experimentation
clawrus group clone backend-agents backend-staging
```

Each thread can override the group's model, thinking mode, timeout, and have its own prompt:

```yaml
groups:
  backend-agents:
    model: claude-sonnet-4-6
    thinking: low
    timeout: 300
    threads:
      - id: "1484056775278989333"
        name: "AuthAgent"
      - id: "1484056779322036234"
        name: "PaymentAgent"
        model: claude-opus-4-6      # per-thread override
        thinking: high
        prompt: "Integrate Stripe checkout for premium tier"
```

**Priority resolution:** CLI flag > per-thread override > group default > hardcoded default.

## Presets (@alias)

Presets compose multiple groups into a single target with automatic thread deduplication.

```bash
# Create a preset from two groups
clawrus preset new deploy backend-agents frontend-agents

# Run against the preset (@ prefix)
clawrus run @deploy "prepare for release v2.1"

# The magic @all preset auto-generates from all known groups
clawrus run @all "health check"
```

If a thread appears in multiple groups within a preset, it receives the message only once.

## Broadcast vs Gather

### Broadcast (default)

Sends the message to all threads in parallel and reports delivery status.

```bash
clawrus run my-agents "implement the new auth flow"
```

```
THREAD          STATUS  ERROR
AuthAgent       OK
PaymentAgent    OK
SearchAgent     FAIL    timeout after 300s
```

### Gather

Sends the message, polls each thread for a reply, then summarizes all responses via LLM.

```bash
clawrus run my-agents "what's your current status?" --mode gather --gather-timeout 90
```

```
Clawrus -- Gather Results
Group: my-agents | Mode: gather | Threads: 3

[AuthAgent] "OAuth integration complete, PR ready for review"
[PaymentAgent] "Stripe webhook handler at 80%, ETA 2 hours"
[SearchAgent] "Elasticsearch indexing running, 3/5 indices rebuilt"

Summary: AuthAgent is done with OAuth (PR ready). PaymentAgent is 80% through
Stripe webhooks (2h ETA). SearchAgent is rebuilding indices (3/5 complete).
```

If the LLM summarization endpoint isn't available, raw replies are printed instead — not an error.

## Per-Thread Prompts

Give each agent in a group its own instructions with `--prompt`:

```bash
clawrus group add sprint-1 1111 --name "AuthAgent"  --prompt "Build SSO with SAML"
clawrus group add sprint-1 2222 --name "PayAgent"   --prompt "Add Stripe billing"
clawrus group add sprint-1 3333 --name "SearchAgent" --prompt "Migrate to Meilisearch"

# Each agent gets its own prompt — no message argument needed
clawrus run sprint-1
```

Update a prompt later:

```bash
clawrus group set-prompt sprint-1 AuthAgent "Switch SSO to OpenID Connect"
```

## Ad-Hoc Runs

Skip groups entirely with `--threads`:

```bash
clawrus run --threads 1111,2222,3333 "emergency: rollback to v2.0"
```

## Zero-Config Gateway Discovery

Clawrus finds the OpenClaw gateway automatically:

1. `--gateway-url` flag (if provided)
2. `~/.clawrus/config.yaml` -> `gateway.url`
3. `http://127.0.0.1:18789` (OpenClaw default port)
4. Scans ports 3000, 8080, 3260

Auth token is read from `OPENCLAW_TOKEN` env var or `~/.openclaw/openclaw.json`.

## Config Files

| File | Purpose |
|------|---------|
| `~/.clawrus/groups.yaml` | Thread groups (main config) |
| `~/.clawrus/presets.yaml` | Named presets (managed by `preset` commands) |
| `~/.clawrus/config.yaml` | Gateway URL override (optional) |
| `~/.openclaw/openclaw.json` | Auth token (read-only, managed by OpenClaw) |

## Documentation

- [Full Command Reference](docs/COMMANDS.md)
- [Usage Guide](docs/GUIDE.md)
- [Architecture](docs/ARCHITECTURE.md)

## License

MIT
