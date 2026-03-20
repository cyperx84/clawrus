# Clawrus

Agent thread orchestration for OpenClaw

## Setup

Requires OpenClaw running locally (`openclaw gateway status`).
No configuration needed — clawrus auto-discovers the gateway.

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
clawrus group add product-ideas 1484056775278989333 --name "AgentPulse"

# Add a thread with a per-thread prompt
clawrus group add my-sprint 1484056775 --name AgentPulse --prompt "Add Supabase auth — email + GitHub OAuth"
clawrus group add my-sprint 1484056779 --name SkillVault --prompt "Add Stripe checkout for premium skills"

# Run with per-thread prompts (each thread gets its own prompt)
clawrus run my-sprint
# → AgentPulse: "Add Supabase auth — email + GitHub OAuth"
# → SkillVault: "Add Stripe checkout for premium skills"

# Update a thread's prompt
clawrus group set-prompt my-sprint AgentPulse "Implement SSO with SAML"

# Remove a thread
clawrus group remove product-ideas 1484056775278989333
```

## Modes

### broadcast (default)

Sends the command to all threads in parallel and reports success/failure for each.

```bash
clawrus run my-group "do the thing"
```

### gather

Sends the command, then polls each thread for a reply. Collects all replies and summarizes them via OpenClaw's built-in LLM routing (`/api/ai/complete`).

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

If `/api/ai/complete` is not available, raw replies are printed instead (not an error).

## Gateway Discovery

Clawrus auto-discovers the OpenClaw gateway in this order:

1. `--gateway-url` CLI flag
2. `~/.clawrus/config.yaml` → `gateway.url` field
3. Auto-detect `http://127.0.0.1:18789` (OpenClaw default)
4. Scan common ports (3000, 8080, 3260)
5. Error with helpful message if nothing found

## Config

`~/.clawrus/groups.yaml`:

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
        prompt: "Add Stripe checkout for premium skills"  # per-thread prompt
      - id: "1484056767544688690"
        name: "ClawApps"
      - id: "1484056771092938773"
        name: "FleetControl"
```

Optional `~/.clawrus/config.yaml` (only needed to override gateway URL):

```yaml
gateway:
  url: http://127.0.0.1:18789
```

## Priority (model/thinking/timeout)

CLI flags > per-thread override > group default > hardcoded default
