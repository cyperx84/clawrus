# ~~Chorus~~ Clawrus 🎵

Agent thread orchestration for OpenClaw

## Install

```bash
go install github.com/cyperx84/clawrus@latest
```

## Quick Start

```bash
# Create sample config
clawrus init

# Edit config with your thread IDs
vim ~/.clawrus/groups.yaml

# List groups
clawrus list

# Send a command to all threads (broadcast mode, default)
clawrus run product-ideas "add auth flow"

# Gather mode: send, collect replies, and summarize
clawrus run product-ideas "status update" --mode gather

# Gather with custom timeout
clawrus run product-ideas "report progress" --mode gather --gather-timeout 90

# Override model for the whole run
clawrus run product-ideas "ship it" --model claude-opus-4-6

# Show group details
clawrus show product-ideas

# Add a thread to a group
clawrus add product-ideas 1484056775278989333 --name "AgentPulse"

# Remove a thread
clawrus remove product-ideas 1484056775278989333
```

## Modes

### broadcast (default)

Sends the command to all threads in parallel and reports success/failure for each.

```bash
clawrus run my-group "do the thing"
```

### gather

Sends the command, then polls each thread for a reply. Collects all replies and summarizes them via an LLM (OpenRouter or OpenAI).

```bash
clawrus run my-group "what's your status?" --mode gather
```

Output:

```
🎵 Clawrus — Gather Results
Group: my-group | Mode: gather | Threads: 4

[AgentPulse] "All tasks complete, ready for review"
[SkillVault] "Processing batch 3 of 5"

📋 Summary: AgentPulse is done and ready for review. SkillVault is still processing (batch 3/5).
```

Set `OPENROUTER_API_KEY` or `OPENAI_API_KEY` for LLM summarization. Without a key, raw replies are printed.

## Config

`~/.clawrus/groups.yaml` (or `CLAWRUS_CONFIG` env):

```yaml
groups:
  product-ideas:
    model: claude-sonnet-4-6
    thinking: low
    timeout: 300
    threads:
      - id: "1484056775278989333"
        name: "AgentPulse"
      - id: "1484056779322036234"
        name: "SkillVault"
        model: glm-5-turbo    # per-thread override
      - id: "1484056767544688690"
        name: "ClawApps"
      - id: "1484056771092938773"
        name: "FleetControl"
```

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENCLAW_URL` | `http://localhost:3260` | OpenClaw gateway URL |
| `OPENCLAW_API_KEY` | | Gateway API key |
| `OPENCLAW_AGENT_ID` | | Target agent ID |
| `CLAWRUS_CONFIG` | `~/.clawrus/groups.yaml` | Config file path |
| `OPENROUTER_API_KEY` | | OpenRouter API key (for gather summaries) |
| `OPENAI_API_KEY` | | OpenAI API key (fallback for gather summaries) |

## Priority (model/thinking/timeout)

CLI flags > per-thread override > group default > hardcoded default
