# AGH — Agent Operating System

AGH is a local-first daemon that manages AI agent sessions through the [Agent Client Protocol (ACP)](https://github.com/coder/acp-go-sdk). It spawns ACP-compatible agents (Claude Code, Codex, Gemini CLI, etc.) as subprocesses, communicates via JSON-RPC over stdio, persists all events in SQLite, and exposes interfaces through HTTP/SSE (web UI) and Unix domain sockets (CLI).

Single binary. No sidecars. No external services.

## Features

- **Multi-agent management** — spawn, monitor, and interact with multiple AI agents simultaneously
- **ACP protocol** — standardized communication with any ACP-compatible agent via JSON-RPC over stdio
- **Session lifecycle** — full state machine (`starting` → `active` → `stopping` → `stopped`) with resume support
- **Event persistence** — per-session SQLite databases with indexed events, turn history, and token usage tracking
- **Observability** — global event aggregation, health checks, token cost tracking, and transcript retention
- **Permission system** — configurable policies (`deny-all`, `approve-reads`, `approve-all`) for agent file access
- **Memory system** — dual-scope persistent memory (global + workspace) with dream consolidation
- **Web UI** — React 19 SPA with real-time SSE streaming
- **CLI** — full-featured Cobra command tree with human, JSON, and compact output formats
- **Dual transport** — HTTP/SSE for the web UI, Unix domain socket for the CLI

## Architecture

```
┌──────────┐     ┌──────────┐
│  Web UI  │     │   CLI    │
│ (React)  │     │ (Cobra)  │
└────┬─────┘     └────┬─────┘
     │ HTTP/SSE       │ UDS
     ▼                ▼
┌──────────────────────────────────────┐
│              daemon/                 │  ← composition root
│         (wires all packages)         │
├──────────────────────────────────────┤
│  api/httpapi  │  api/udsapi          │
│  (HTTP/SSE)   │  (Unix socket)       │
│               └── api/core (shared)  │
├──────────┬───────────────────────────┤
│ session/ │  observe/  │  memory/     │
│ (Manager)│  (metrics) │  (dual-scope)│
├──────────┼────────────┴──────────────┤
│  acp/    │  store/    │  workspace/  │
│ (Driver) │ (SQLite)   │  skills/     │
└──────────┴────────────┴──────────────┘
         │
    JSON-RPC/stdio
         │
    ┌────┴────┐
    │  Agent  │  (Claude Code, Codex, Gemini CLI, ...)
    └─────────┘
```

**Package boundaries are strict** — dependencies flow downward only. `daemon/` is the sole composition root; no package imports it. Interfaces are defined where consumed (Go-style). No event bus — direct function calls through typed interfaces.

## Quick Start

### Prerequisites

- Go 1.25+
- [Bun](https://bun.sh) (for web UI)
- At least one ACP-compatible agent installed (e.g., Claude Code)

### Build

```bash
make build          # compile binary → bin/agh
make verify         # full gate: web lint/typecheck/test/build → Go fmt/lint/test/build
```

### Run

```bash
# Bootstrap AGH for the current user
agh install

# Start the daemon (detaches to background)
agh daemon start

# Check status
agh daemon status

# Create a session with the default agent
agh session new

# Send a prompt
agh session prompt <session-id> "Explain this codebase"

# Stream events in real-time
agh session events <session-id> --follow

# Stop the daemon
agh daemon stop
```

Then open `http://localhost:2123` in your browser.

## CLI Commands

```
agh
├── install                         # Bootstrap config + default agent
├── version                         # Print version
├── daemon
│   ├── start [--foreground]        # Start daemon
│   ├── stop                        # Stop daemon
│   └── status                      # Show daemon status
├── session
│   ├── new [--agent <name>]        # Create session
│   ├── list [--all]                # List sessions
│   ├── status <id>                 # Session details
│   ├── stop <id>                   # Stop session
│   ├── resume <id>                 # Resume session
│   ├── wait <id>                   # Block until stopped
│   ├── prompt <id> <message>       # Send prompt
│   ├── events <id> [--follow]      # Query/stream events
│   └── history <id>                # Turn-grouped history
├── agent
│   ├── list                        # List installed agents
│   └── info <name>                 # Agent details
├── memory
│   ├── list [--scope]              # List memories
│   ├── read <file>                 # Read memory
│   ├── write                       # Write memory
│   ├── delete <file>               # Delete memory
│   └── consolidate                 # Trigger dream consolidation
├── observe
│   ├── health                      # Daemon health
│   ├── events                      # Event summaries
│   └── reconcile                   # Reconcile state
└── whoami                          # Connected user info
```

All commands support `--output human|json|toon`.

## Configuration

AGH uses TOML configuration with global + workspace overlay:

- **Global**: `~/.agh/config.toml`
- **Workspace**: `.agh/config.toml` (merged on top)

```toml
[daemon]
socket = "~/.agh/daemon.sock"

[http]
host = "localhost"
port = 2123

[defaults]
agent = "general"
provider = "claude"

[limits]
max_sessions = 10
max_concurrent_agents = 20

[permissions]
mode = "approve-all"      # deny-all | approve-reads | approve-all

[providers.claude]
default_model = "claude-sonnet-4-20250514"
api_key_env = "ANTHROPIC_API_KEY"

[providers.codex]
default_model = "gpt-4o"
api_key_env = "OPENAI_API_KEY"

[providers.gemini]
default_model = "gemini-2.5-pro"
api_key_env = "GEMINI_API_KEY"

[observability]
enabled = true
retention_days = 7

[log]
level = "info"    # debug | info | warn | error
```

## Agent Definitions

Agents are defined via `AGENT.md` files with YAML frontmatter:

```yaml
name: claude
provider: claude
model: claude-sonnet-4-20250514
tools: [read, glob, grep, write, bash]
permissions: approve-all
```

`provider` and `model` may be omitted when the user-global defaults written by `agh install` should be used.

## Web UI

The web frontend is a React 19 SPA built with Vite, TanStack Router/Query, Tailwind CSS v4, and shadcn/ui. In normal daemon operation, the SPA is served by the daemon itself on the configured HTTP host/port; `make web-dev` remains the separate Vite workflow for frontend iteration.

```bash
make web-dev        # Dev server on :3000 (proxies API to :2123)
make web-build      # Production build consumed by the daemon binary
make web-lint       # oxfmt + oxlint
make web-test       # Vitest
```

## Project Structure

```
cmd/agh/                    # CLI entry point
internal/
├── acp/                    # ACP client: subprocess spawn, JSON-RPC over stdio
├── session/                # Session lifecycle, Manager, state machine
├── store/                  # SQLite shared helpers, schema, validation
│   ├── globaldb/           # Global catalog (agh.db)
│   └── sessiondb/          # Per-session event store (events.db)
├── config/                 # TOML loading, validation, agent definition parsing
├── daemon/                 # Composition root: boot, lock, shutdown
├── api/
│   ├── contract/           # Shared daemon/CLI/HTTP contract types
│   ├── core/               # Shared handler types, error mapping, SSE helpers
│   ├── httpapi/            # HTTP/SSE server (Gin)
│   ├── udsapi/             # Unix domain socket server
│   └── testutil/           # Test helpers for the API layer
├── cli/                    # Cobra command tree
├── observe/                # Event recording, health metrics, query engine
├── memory/                 # Persistent dual-scope memory (global + workspace)
│   └── consolidation/      # Dream consolidation runtime
├── skills/                 # Skills catalog and loader
│   └── bundled/            # Bundled skill definitions
├── workspace/              # Workspace resolver and entity management
├── transcript/             # Canonical message assembly from events
├── frontmatter/            # YAML frontmatter parsing
├── fileutil/               # Shared filesystem helpers
├── filesnap/               # File snapshot utilities
├── procutil/               # Process utilities
├── testutil/               # Shared test helpers
├── logger/                 # Structured logging (slog)
└── version/                # Build metadata
web/                        # React 19 SPA
docs/                       # Design documents and plans
```

## Development

```bash
make verify         # Full CI gate (fmt → lint → test → build)
make fmt            # Format Go code
make lint           # golangci-lint (zero tolerance)
make test           # Unit tests with -race
make test-integration  # All tests including integration
make deps           # go mod tidy
make build          # Compile to bin/agh
```

## Roadmap

1. **Phase 1** — Agent core (current): daemon, sessions, ACP, persistence, CLI, web UI
2. **Phase 2** — Memory, skills, and state layers
3. **Phase 3** — Agent network protocol

## License

Private — all rights reserved.
