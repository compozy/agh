# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 implementation is complete and verified; `internal/config`, `internal/logger`, and `internal/version` now exist with tests and example config.
- Task 02 implementation is complete and verified; `internal/store` now provides per-session and global SQLite storage plus atomic session metadata I/O.
- Task 03 implementation is complete and verified; `internal/acp` now provides subprocess launch, prompt streaming, permission handling, and resume-aware ACP session startup.
- Task 04 implementation is complete and verified; `internal/session` now owns active session lifecycle, per-session event writes, atomic meta state transitions, prompt fan-out, and crash-to-stopped handling.
- Task 05 implementation is complete and verified; `internal/observe` now records global event summaries, token aggregates, and permission audit rows, exposes health/query helpers, and reconciles session metadata into `agh.db`.
- Task 06 implementation is complete and verified; `internal/daemon` now owns lock/info lifecycle, boot-time reconciliation, graceful shutdown ordering, signal handling, and the composition-root wiring for `store`, `observe`, and `session`.
- Task 07 implementation is complete and verified; `internal/udsapi` now exposes the daemon API over `daemon.sock`, with Gin-backed request/response routes, SSE follow/wait streams, real Unix-socket integration tests, and default daemon wiring.
- Task 08 implementation is complete and verified; `internal/cli` now provides the Cobra CLI over the task 07 UDS API, including daemon/session/agent/observe/whoami command groups, detached daemon start, human/json/toon output modes, relative `--since` parsing, and CLI integration tests against a real UDS-backed test daemon.
- Task 09 implementation is complete and verified; `internal/httpapi` now exposes the daemon over real TCP HTTP/SSE with Gin middleware, AI SDK-compatible prompt streaming, session/cross-session replay streams, HTTP integration tests, and default daemon wiring.

## Shared Decisions
- Provider resolution is exposed through `Config.ResolveProvider` and `Config.ResolveAgent`, using the precedence chain from the tech spec.
- Config loading reads workspace `.env` before resolving `AGH_HOME`, then merges built-in defaults, global `~/.agh/config.toml`, and workspace `.agh/config.toml`.
- Store timestamps are persisted as fixed-width UTC text so SQLite text ordering is stable for `since`, `limit`, and session listing queries without custom time adapters.
- Per-session event writes go through a dedicated writer goroutine with the next sequence seeded from `MAX(sequence)` on open; later tasks should rely on query order and `AfterSequence` rather than deriving sequence state elsewhere.
- `internal/acp` uses the lower-level `acp.Connection` APIs instead of `ClientSideConnection` because `acp-go-sdk` v0.6.3 does not expose typed `PromptResponse.usage` or `usage_update` fields; downstream tasks should treat usage as opportunistic raw JSON decoded inside the ACP package.
- `internal/session` defines its own `AgentDriver` interface and `AgentProcess` wrapper, then adapts `internal/acp.Driver` through `NewACPDriverAdapter`; downstream packages can depend on the session interface without constructing ACP internals directly in tests.
- `internal/session` keeps only active sessions in-memory plus pending reservations for limit enforcement; stopped sessions are rehydrated from `meta.json` and the existing `events.db`.
- `internal/udsapi` serves follow/reconnect endpoints by polling persisted SQLite rows instead of depending on live notifier fan-out, so SSE replay ids stay aligned with the durable store.
- `internal/session` now exposes `ListAll`, `Status`, `Events`, and `History` helpers for transport/query layers and persists a `session_stopped` event so wait/follow can terminate from stored state.

## Shared Learnings
- Tracking checklist and workflow-memory references live under the skill directories in `.agents/skills/.../references/`.
- The repo’s verification gate is `make verify`; a separate `go test -cover ./...` run is still needed to prove the coverage requirement for task closeout.
- The SDK dispatches inbound ACP notifications concurrently, so `internal/acp` drains a short prompt-stream quiescence window before emitting the final `done` event to avoid dropping trailing `session/update` notifications.
- `internal/session` achieved the 80% coverage target only after explicitly testing cleanup/error/helper branches; future runtime tasks should budget time for those paths instead of assuming lifecycle happy-path tests are enough.
- `internal/observe` relies on `session.Notifier` ordering plus a small in-memory session cache to attach `agent_name` and resolved permission mode to live global audit rows; downstream daemon/API tasks should preserve that wiring rather than duplicating session lookups.
- `internal/daemon` shares a single `store.GlobalDB` instance with `observe.New(...)`, fans out `session.Notifier` callbacks through a daemon-owned multi-notifier, and uses injectable server factories so later HTTP/UDS tasks can plug in without widening task 06 scope.
- `internal/httpapi` remains separate from `internal/udsapi` for now; both expose the same route contract, but only HTTP task 09 adds the AI SDK UI message stream translation and CORS/logging middleware.
- Unix socket paths in tests need to stay short on macOS; using the raw `t.TempDir()` socket path can exceed the platform limit and fail `bind`.
- Integration-only helper functions should live in `//go:build integration` test files when they are not used by the unit-test build; otherwise `golangci-lint` reports them as unused during `make verify`.

## Open Risks
- No functional consumers of the new config/logger APIs exist yet, so downstream tasks should validate integration points when wiring daemon/session packages.

## Handoffs
- Later tasks can rely on the new provider registry, home path layout helpers, AGENT.md parser, and logger constructor instead of re-implementing those concerns.
- Later tasks can use `OpenSessionDB`, `OpenGlobalDB`, `ReadSessionMeta`, and `WriteSessionMeta` instead of re-implementing SQLite setup, corruption recovery, or atomic metadata writes.
- Later tasks can use `session.NewManager(...)` with `WithHomePaths`, `WithConfig/WithConfigLoader`, `WithNotifier`, and the default `ACPDriverAdapter` wiring instead of rebuilding lifecycle orchestration.
- Later tasks can use `observe.New(...)`, `Observer.Health(...)`, `Observer.QueryEvents(...)`, `Observer.QueryTokenStats(...)`, `Observer.QueryPermissionLog(...)`, and `Observer.Reconcile(...)` instead of querying the global DB directly.
- Later tasks can use `daemon.New(...)`, `daemon.WithHTTPServerFactory(...)`, and `daemon.WithUDSServerFactory(...)` to attach concrete transport layers while preserving the daemon as the sole composition root.
- Later CLI work can rely on `internal/udsapi` already serving the techspec endpoints over `daemon.sock`; follow/wait should use those SSE endpoints instead of inventing a separate IPC protocol.
- `cmd/agh` now delegates to `internal/cli.ExecuteContext`, so future CLI tasks should extend the Cobra command tree rather than reintroducing ad-hoc argv parsing in `cmd/agh/main.go`.
- Later work can use `internal/httpapi` for browser/web transport needs and `internal/udsapi` for local CLI transport needs; if route duplication becomes a maintenance problem, extract shared handler logic as a follow-up instead of pushing that refactor into unrelated tasks.
