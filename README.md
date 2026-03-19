# ThreadGroups (`tg`)

Orchestrate commands across groups of OpenClaw Discord threads.

## Install

```bash
go install github.com/cyperx84/threadgroups@latest
```

## Quick Start

```bash
# Create sample config
tg init

# Edit config with your thread IDs
vim ~/.threadgroups/groups.yaml

# List groups
tg list

# Send a command to all threads in a group
tg run product-ideas "add auth flow"

# Override model for the whole run
tg run product-ideas "ship it" --model claude-opus-4-6

# Show group details
tg show product-ideas

# Add a thread to a group
tg add product-ideas 1484056775278989333 --name "AgentPulse"

# Remove a thread
tg remove product-ideas 1484056775278989333
```

## Config

`~/.threadgroups/groups.yaml` (or `THREADGROUPS_CONFIG` env):

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
| `THREADGROUPS_CONFIG` | `~/.threadgroups/groups.yaml` | Config file path |

## Priority (model/thinking/timeout)

CLI flags > per-thread override > group default > hardcoded default
