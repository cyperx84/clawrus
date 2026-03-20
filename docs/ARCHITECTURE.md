# Architecture

Internals of clawrus for contributors and AI agents that need to understand how things work.

---

## Overview

```
┌─────────────┐     HTTP POST      ┌──────────────────┐     Discord API     ┌─────────────┐
│  clawrus    │ ──────────────────> │  OpenClaw Gateway │ ─────────────────> │ Agent Thread │
│  (CLI)      │  /tools/invoke      │  (localhost)      │                    │ (Discord)    │
└─────────────┘                     └──────────────────┘                     └─────────────┘
```

Clawrus is a stateless CLI. It reads config from YAML files, sends HTTP requests to the OpenClaw gateway, and prints results. It does not maintain connections, daemons, or background processes.

---

## Project Structure

```
clawrus/
├── main.go                    # Entry point — calls cli.RootCmd().Execute()
├── internal/
│   ├── cli/cli.go             # All Cobra commands and run logic
│   ├── config/config.go       # YAML config loading and saving
│   ├── gateway/gateway.go     # HTTP client for OpenClaw gateway
│   └── types/types.go         # Shared data structures
├── .goreleaser.yml            # Build matrix (darwin/linux, amd64/arm64)
└── .github/workflows/
    ├── ci.yml                 # Build + test on push/PR
    └── release.yml            # GoReleaser on tag push
```

---

## Gateway Auto-Discovery

Implemented in `internal/gateway/gateway.go` — `DiscoverGateway(flagURL, configURL)`.

**Priority order:**

1. `--gateway-url` CLI flag
2. `gateway.url` from `~/.clawrus/config.yaml`
3. Ping `http://127.0.0.1:18789` (OpenClaw default port)
4. Scan ports 3000, 8080, 3260 with HTTP ping
5. Return error with suggestion to run `openclaw gateway status`

The ping check hits `GET {url}/` and accepts any 2xx response.

### Auth Token Discovery

`DiscoverAuthToken()` in the same file:

1. `OPENCLAW_TOKEN` environment variable
2. Parse `~/.openclaw/openclaw.json` and extract `gateway.auth.token`
3. Return empty string (no auth — gateway may not require it)

The token is sent as `Authorization: Bearer <token>` on all gateway requests.

---

## Gateway Communication

All agent interactions go through a single endpoint: `POST /tools/invoke`.

### Sending a Message

```go
// gateway.SendMessage(threadID, message, model, thinking, timeout)

POST {gateway}/tools/invoke
Content-Type: application/json
Authorization: Bearer <token>

{
    "tool": "message",
    "args": {
        "action": "send",
        "channel": "discord",
        "target": "<thread-id>",
        "message": "<message>"
    }
}
```

**Optional fields in `args`** (included only when non-empty):

| Field | Source | Description |
|-------|--------|-------------|
| `model` | Resolved from priority chain | Model to use for this thread |
| `thinking` | Resolved from priority chain | Thinking mode for this thread |
| `timeout` | Resolved from priority chain | Timeout in seconds |

**Response:**

```json
{"ok": true, "status": "sent"}
```

Or on failure:

```json
{"ok": false, "error": "thread not found"}
```

### Reading Replies (Gather Mode)

```go
// gateway.PollReply(threadID, gatherTimeout)

POST {gateway}/tools/invoke
Content-Type: application/json
Authorization: Bearer <token>

{
    "tool": "message",
    "args": {
        "action": "read",
        "channel": "discord",
        "target": "<thread-id>",
        "limit": 5,
        "after": "<message-id>"     // optional, for pagination
    }
}
```

**Response:** JSON array of message objects. The poller looks for the first entry with a non-empty `content` field.

**Polling loop:** Every 3 seconds until `gatherTimeout` is reached. Returns `"no reply within timeout"` if nothing arrives.

### LLM Summarization

```go
// gateway.SummarizeReplies(replies)

POST {gateway}/api/ai/complete
Content-Type: application/json

{
    "prompt": "Summarize these agent replies into a concise unified status:\n<replies>",
    "model": "glm-5-turbo"
}
```

**Response parsing** tries these fields in order: `content`, `text`, `choices[0].message.content`.

Returns empty string on 404 (endpoint not available) — this is expected and not an error.

---

## Data Models

Defined in `internal/types/types.go`:

```go
type GroupConfig struct {
    Groups map[string]Group `yaml:"groups"`
}

type Group struct {
    Model    string   `yaml:"model,omitempty"`
    Thinking string   `yaml:"thinking,omitempty"`
    Timeout  int      `yaml:"timeout,omitempty"`
    Threads  []Thread `yaml:"threads"`
}

type Thread struct {
    ID       string `yaml:"id"`
    Name     string `yaml:"name,omitempty"`
    Model    string `yaml:"model,omitempty"`
    Thinking string `yaml:"thinking,omitempty"`
    Timeout  int    `yaml:"timeout,omitempty"`
    Prompt   string `yaml:"prompt,omitempty"`
}

type PresetConfig struct {
    Presets map[string]Preset `yaml:"presets"`
}

type Preset struct {
    Groups []string `yaml:"groups"`
}

type RunResult struct {
    ThreadID   string
    ThreadName string
    OK         bool
    Error      string
    Reply      string   // populated only in gather mode
}

type ClawrusConfig struct {
    Gateway GatewayConfig `yaml:"gateway"`
}

type GatewayConfig struct {
    URL string `yaml:"url"`
}
```

---

## Config File Loading

All in `internal/config/config.go`:

| Function | File | Behavior on missing |
|----------|------|---------------------|
| `Load()` | `~/.clawrus/groups.yaml` | Error (required) |
| `LoadPresets()` | `~/.clawrus/presets.yaml` | Empty map (optional) |
| `LoadMainConfig()` | `~/.clawrus/config.yaml` | Empty struct (optional) |

The `CLAWRUS_CONFIG` env var overrides the path for `groups.yaml`.

---

## Priority Resolution

Model, thinking mode, and timeout all follow the same priority chain:

```
CLI flag  >  per-thread override  >  group default  >  hardcoded default
```

Implemented via `resolveModel()`, `resolveThinking()`, `resolveTimeout()` in `cli.go`:

```go
func resolveModel(groupModel, threadModel, flagModel string) string {
    if flagModel != "" {
        return flagModel
    }
    if threadModel != "" {
        return threadModel
    }
    return groupModel
}
```

Timeout defaults to 300 seconds if nothing else is set.

---

## Concurrency Model

### Broadcast

```go
sem := make(chan struct{}, flagParallel)  // default: 4
var wg sync.WaitGroup

for _, thread := range threads {
    wg.Add(1)
    go func(t Thread) {
        defer wg.Done()
        sem <- struct{}{}        // acquire
        defer func() { <-sem }() // release

        result := gw.SendMessage(t.ID, message, model, thinking, timeout)
        mu.Lock()
        results = append(results, result)
        mu.Unlock()
    }(thread)
}

wg.Wait()
```

### Gather Polling

After broadcast, a second parallel pass polls for replies:

```go
for _, r := range results {
    if r.OK {
        wg.Add(1)
        go func(result *RunResult) {
            defer wg.Done()
            reply := gw.PollReply(result.ThreadID, gatherTimeout)
            result.Reply = reply
        }(r)
    }
}
wg.Wait()
```

Polling is unbounded (no semaphore) since it's lightweight read operations.

---

## Preset Resolution

`resolvePreset()` in `cli.go`:

1. Load `~/.clawrus/presets.yaml`
2. Strip `@` prefix from name, look up preset
3. If not found and name is `"all"`, auto-generate from all groups
4. For each group in the preset, load its threads
5. Deduplicate by thread ID using `map[string]bool`
6. Return a synthetic `Group` with merged threads

```go
seen := map[string]bool{}
var merged []Thread

for _, groupName := range preset.Groups {
    group := cfg.Groups[groupName]
    for _, t := range group.Threads {
        if !seen[t.ID] {
            seen[t.ID] = true
            merged = append(merged, t)
        }
    }
}
```

---

## Build and Release

### GoReleaser

- **Targets:** darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
- **CGO:** Disabled (`CGO_ENABLED=0`)
- **Version injection:** `-X github.com/cyperx84/clawrus/internal/cli.Version={{.Version}}`
- **Archives:** `clawrus_<version>_<os>_<arch>.tar.gz`
- **Checksums:** SHA256

### CI Pipeline

- **ci.yml:** Runs `go build` and `go test` on every push/PR to main
- **release.yml:** Triggered on `v*` tags, runs GoReleaser, updates Homebrew formula in `cyperx84/homebrew-tap`

### Homebrew

```bash
brew install cyperx84/tap/clawrus
```

The formula is auto-updated by the release workflow.
