# CLAUDE.md

## Project Overview

AGH is an Agent Operating System — a Go single-binary daemon that manages AI agent sessions via ACP (Agent Client Protocol). It spawns ACP-compatible agents (Claude Code, Codex, Gemini CLI, etc.) as subprocesses, communicates via JSON-RPC over stdio, persists events in SQLite, and exposes interfaces via HTTP/SSE (web UI) and UDS (CLI).

**Goals**: daemon single-binary in background, strong observability, agent-first system (agents manipulate via CLI + REST), highly configurable.

**Phases**: 1) Agent core (current) → 2) Memory/Skills/State layers → 3) Agent network protocol

## Critical Rules

- **`make verify` MUST pass** before completing ANY task (runs `fmt → lint → test → build`). Zero warnings, zero errors. No exceptions.
- **`make lint` has zero tolerance** — any golangci-lint issue is a blocking failure
- **Check dependent package APIs** before writing integration code or tests
- **Never add dependencies by hand in `go.mod`** — always use `go get`
- **Never use web search tools for local project code** — use Grep/Glob instead. Web search is only for external docs.
- **Never run destructive git commands** (`git restore`, `git checkout`, `git reset`, `git clean`, `git rm`) **without explicit user permission**. If the worktree contains unexpected edits, read and work around them.

## Skill Dispatch

Activate skills **before** writing code. Match task domain → activate all required skills:

| Domain                  | Required Skills                           | Conditional Skills      |
| ----------------------- | ----------------------------------------- | ----------------------- |
| Go / Runtime            | `golang-pro`                              | `context7`              |
| Config / Logging        | `golang-pro`                              |                         |
| Bug fix                 | `systematic-debugging` + `no-workarounds` | `testing-anti-patterns` |
| Writing Go tests        | `testing-anti-patterns` + `golang-pro`    |                         |
| Task completion         | `verification-before-completion`          |                         |
| Architecture audit      | `architectural-analysis`                  | `adversarial-review`    |
| Creative / new features | `brainstorming`                           |                         |
| Git rebase/conflicts    | `git-rebase`                              |                         |

Web skill dispatch is in `web/CLAUDE.md`.

Every domain change requires its skill — no skipping "because it's a small change". Activate multiple skills when code touches multiple domains.

## Build Commands

### Go (backend)

```bash
make verify              # BLOCKING GATE: fmt → lint → test → build
make fmt                 # Format with gofmt
make lint                # Strict golangci-lint (zero issues)
make test                # Run tests with -race flag
make build               # Compile binary
make deps                # Tidy and verify modules
```

## Commit style: <type>: <description>

## Code Search Hierarchy

1. **Grep / Glob** — for local project code
2. **`context7` skill** — for external library documentation
3. **Web search tools** — for web research, news, external code examples

## Old Project Reference

The `.old_project/` directory contains the previous AGH implementation (78K+ LOC). **Reference only** — do not modify, do not import, do not include in builds. Exclude from code search results.

## Architecture

### Principles

- **Designed for incremental extension** — new capabilities arrive as new packages wired into `daemon/`, without modifying existing packages. Small interfaces + dependency injection.
- **Pragmatic Flat with Discipline** — 11 packages under `internal/`, no domain/infra split, no event bus
- **`daemon/` is the sole composition root** — the only package that imports all others
- **No package imports `daemon/`, `httpapi/`, `udsapi/`, or `cli/`** — dependencies flow downward only
- **Interfaces defined where consumed** (Go-style) — `session/` defines `AgentDriver`, `acp/` implements it
- **Direct function calls through interfaces** — no event bus, no NATS, no reflection-based routing
- **Notifier pattern for fan-out** — typed interface for observability and SSE, not a generic bus
- **No back-pointers between packages** — inject callbacks or interfaces
- **Functional options for constructors** — `NewManager(opts ...Option)`
- **Maps for <10 items** — no registry interfaces for small collections
- **File-level organization** within packages — sub-packages only when complexity justifies it
- **CI-enforceable boundaries** — grep rules to prevent import cycles

### Concurrency

- Every goroutine must have explicit ownership and shutdown via `context.Context` cancellation
- No fire-and-forget goroutines — track with `sync.WaitGroup` or equivalent
- Use `select` with `ctx.Done()` in all long-running goroutine loops
- Prefer channels over shared memory with mutexes when practical
- `sync.RWMutex` for read-heavy, `sync.Mutex` for write-heavy shared state
- No `time.Sleep()` in orchestration — use proper synchronization primitives

### Runtime

- Single-binary and local-first. Sidecars or external control planes require a written techspec.
- Keep execution paths deterministic and observable.

## Package Layout

| Path               | Responsibility                                                 |
| ------------------ | -------------------------------------------------------------- |
| `cmd/agh`          | Main entry point, CLI binary                                   |
| `internal/config`  | TOML loading, validation, merge, home paths, agent def parsing |
| `internal/acp`     | ACP client: subprocess spawn, JSON-RPC over stdio              |
| `internal/session` | Session lifecycle, Manager, state machine                      |
| `internal/store`   | SQLite (per-session events.db + global agh.db)                 |
| `internal/observe` | Event recording, health metrics, query engine                  |
| `internal/httpapi` | HTTP/SSE server (Gin) for web UI                               |
| `internal/udsapi`  | UDS server for CLI IPC                                         |
| `internal/cli`     | Cobra commands                                                 |
| `internal/daemon`  | Composition root, lock, boot, shutdown                         |
| `internal/logger`  | Structured logging (slog)                                      |
| `internal/version` | Build metadata                                                 |
| `web/`             | React 19 SPA (Vite, TanStack Router/Query, Tailwind, shadcn)   |
| `web/src/systems/` | Domain feature modules (app-renderer-systems pattern)          |

## Coding Style

- Explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`
- `errors.Is()` and `errors.As()` for error matching — never compare error strings
- Never ignore errors with `_` — every error must be handled or have a written justification
- No `panic()` or `log.Fatal()` in production paths — only for truly unrecoverable startup failures
- `log/slog` for structured logging — no `log.Printf` or `fmt.Println` for operational output
- `context.Context` as first argument to functions crossing runtime boundaries — avoid `context.Background()` outside `main` and focused tests
- Compile-time interface verification: `var _ Interface = (*Type)(nil)`
- No `interface{}`/`any` when a concrete type is known
- No reflection without performance justification
- Never hardcode configuration — use TOML config or functional options

## Testing

- Table-driven tests with subtests (`t.Run`) as default
- `t.Parallel()` for independent subtests
- `t.TempDir()` for filesystem isolation
- `t.Helper()` on test helper functions
- `-race` flag must pass before committing
- Mock via interfaces, not test-only methods in production code
- **80% coverage minimum** per package

### Integration Tests

- **Build tags**: `//go:build integration` at top of `*_integration_test.go` files
- **Co-located** with the package they test (not in a separate `test/` directory)
- `make test` = unit only. `make test-integration` = everything (`-tags integration`).
- `TestMain` for expensive one-time setup/teardown
- Use **real dependencies** (real SQLite via `t.TempDir()`, mock ACP server as subprocess)
- Keep fast enough for CI (~30s max per package)
