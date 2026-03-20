# Usage Guide

Practical patterns for orchestrating agent fleets with clawrus.

---

## Setting Up Your First Fleet

### 1. Initialize

```bash
clawrus init
```

This creates `~/.clawrus/groups.yaml` with an example structure.

### 2. Create a Group

```bash
clawrus group new my-fleet
```

### 3. Add Agent Threads

Each agent thread in OpenClaw has a Discord thread ID. Add them to your group:

```bash
clawrus group add my-fleet 1484056775278989333 --name "AgentAlpha"
clawrus group add my-fleet 1484056779322036234 --name "AgentBeta"
clawrus group add my-fleet 1484056767544688690 --name "AgentGamma"
```

### 4. Verify

```bash
clawrus show my-fleet
```

### 5. Send Your First Command

```bash
clawrus run my-fleet "hello, report your current status"
```

---

## Orchestrating a Status Check

The most common pattern: ask every agent what it's doing and get a summary.

```bash
clawrus run my-fleet "give me a one-line status update" --mode gather --gather-timeout 90
```

Output:

```
Clawrus -- Gather Results
Group: my-fleet | Mode: gather | Threads: 3

[AgentAlpha] "Auth module complete, writing tests"
[AgentBeta] "Payment integration 60% done"
[AgentGamma] "Search indexing failed, investigating"

Summary: Alpha is done with auth (testing phase). Beta is midway through payments.
Gamma hit an issue with search indexing and is debugging.
```

### Longer Timeouts for Complex Queries

If agents need time to think before replying:

```bash
clawrus run my-fleet "analyze your codebase for security issues" \
  --mode gather \
  --gather-timeout 180 \
  --thinking high
```

---

## Broadcast Patterns

### Deploy Command to All Agents

```bash
clawrus run my-fleet "deploy to staging and report the URL"
```

### Emergency Rollback

```bash
clawrus run my-fleet "URGENT: revert last deployment immediately" --parallel 8
```

### Model Override for Heavy Tasks

```bash
clawrus run my-fleet "refactor the data layer to use repository pattern" \
  --model claude-opus-4-6 \
  --thinking high \
  --timeout 600
```

---

## Run Modes

Clawrus has four run modes — pick based on what you need back.

### Broadcast (default)

Fire-and-forget. Sends to all threads in parallel, prints a pass/fail table. Use for issuing commands, not reading responses.

```bash
clawrus run backend "deploy to staging"
```

### Gather

Sends, waits for replies, then LLM-summarizes. Best for status checks and questions.

Three phases:
1. **Send** — broadcasts to all threads in parallel
2. **Poll** — checks each thread every 3s until `--gather-timeout`
3. **Summarize** — sends collected replies to `/v1/chat/completions` for an LLM summary

```bash
clawrus run backend "status update" --mode gather --gather-timeout 120
```

### Pipeline

Sequential chain: each thread's reply becomes the next thread's input. Use for multi-stage reasoning — research → draft → critique, or plan → implement → review.

```bash
# Create a 3-step chain
clawrus group new chain
clawrus group add chain 1111 --name "Researcher"
clawrus group add chain 2222 --name "Writer"
clawrus group add chain 3333 --name "Critic"

# Run: Researcher gets the prompt, Writer gets Researcher's reply, Critic gets Writer's reply
clawrus run chain "Write a technical doc on rate limiting" --mode pipeline --gather-timeout 120
```

If a step times out, the previous output is passed forward so the chain doesn't break.

### Poll

Quick fleet health sweep. Broadcasts then collects replies into a compact table — no LLM, just raw status.

```bash
clawrus run @ops "quick status?" --mode poll --gather-timeout 30
```

Output:
```
📡 Clawrus — Poll Results | Group: ops | Threads: 3

THREAD                STATUS  REPLY
System Health         ✅      All services nominal, last check 2m ago
Alert & Incident      ✅      No active incidents
Maintenance           ✅      Cleanup job queued for 02:00
```

### When to use what

| Mode | Use when |
|------|----------|
| broadcast | Issuing commands, fire-and-forget |
| gather | Questions, status checks, need a summary |
| pipeline | Multi-stage reasoning chains |
| poll | Fast fleet health check, no summarization needed |

---

## Template Substitution

Any message or per-thread prompt can use placeholders that are filled in at send time:

| Placeholder | Value |
|-------------|-------|
| `{{name}}` | Thread name |
| `{{group}}` | Group name |
| `{{preset}}` | Preset name (empty if none) |
| `{{date}}` | Current date `YYYY-MM-DD` |
| `{{time}}` | Current time `HH:MM` (24h) |

### Personalised prompts

```bash
clawrus group add sprint-7 1111 --name "AuthAgent" \
  --prompt "Hey {{name}}, it's {{date}}. Report progress on your task for group {{group}}."
```

### Dynamic broadcast

```bash
clawrus run @all "Morning standup {{date}}: what did {{name}} ship yesterday?"
```

---

## Per-Thread Prompts for Sprint Planning

Give each agent its own task for a sprint:

```bash
clawrus group new sprint-7

clawrus group add sprint-7 1111 --name "AuthAgent"   --prompt "Build SSO with SAML provider"
clawrus group add sprint-7 2222 --name "PayAgent"    --prompt "Add Stripe billing portal"
clawrus group add sprint-7 3333 --name "SearchAgent" --prompt "Migrate from Elasticsearch to Meilisearch"
```

Kick off the sprint — each agent gets its own prompt:

```bash
clawrus run sprint-7
```

Check on progress:

```bash
clawrus run sprint-7 "what's your progress so far?" --mode gather
```

Update a prompt mid-sprint:

```bash
clawrus group set-prompt sprint-7 AuthAgent "Switch from SAML to OpenID Connect"
```

---

## Named Presets for Daily Workflows

### The All-Hands Check

```bash
clawrus preset new morning-check backend frontend infra
clawrus run @morning-check "good morning, status update please" --mode gather
```

### Deploy Pipeline

```bash
clawrus preset new deploy backend frontend
clawrus run @deploy "run tests, build, and deploy to staging"
```

### The @all Shortcut

The `@all` preset auto-generates from every group in your config:

```bash
clawrus run @all "health check"
```

### Thread Deduplication

If `AgentAlpha` is in both `backend` and `infra`, a preset containing both groups will only message `AgentAlpha` once.

---

## Ad-Hoc Runs

Skip groups entirely when you need a one-off:

```bash
clawrus run --threads 1484056775278989333,1484056779322036234 "quick test"
```

Useful for debugging or testing before committing threads to a group.

---

## Group Management Patterns

### Clone for Experimentation

```bash
clawrus group clone production staging
clawrus group set staging --model claude-haiku-4-5-20251001 --timeout 120
```

### Organize by Concern

```yaml
# ~/.clawrus/groups.yaml
groups:
  backend:
    model: claude-sonnet-4-6
    threads: [...]
  frontend:
    model: claude-sonnet-4-6
    threads: [...]
  infra:
    model: claude-opus-4-6
    thinking: high
    timeout: 600
    threads: [...]
```

### Per-Thread Model Overrides

Some agents need bigger models:

```bash
clawrus group add backend 1111 --name "ArchitectAgent" --model claude-opus-4-6
clawrus group add backend 2222 --name "LintAgent" --model claude-haiku-4-5-20251001
```

---

## Tips and Patterns

**Start small.** Begin with 2-3 agents in a group. Scale up once you trust the flow.

**Use names.** Always `--name` your threads. `AuthAgent` is easier to read than `1484056775278989333` in output tables.

**Gather for decisions, broadcast for commands.** Use gather when you need to read responses. Use broadcast when you're issuing instructions.

**Tune parallelism.** The default `--parallel 4` is conservative. Bump to 8-16 if your gateway handles it.

**Tune timeouts.** Complex tasks need longer timeouts. Use `--timeout 600` for refactoring, `--gather-timeout 120` for detailed status checks.

**Use presets for recurring workflows.** If you run the same set of groups daily, make a preset.

**Per-thread prompts for sprints.** Assign tasks to agents via prompts, then use a single `clawrus run` to kick them all off.
