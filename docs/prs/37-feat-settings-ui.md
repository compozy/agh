# PR #37: feat: settings ui

- **URL**: https://github.com/compozy/agh/pull/37
- **Author**: @pedronauck
- **State**: merged
- **Created**: 2026-04-18T02:01:19Z
- **Merged**: 2026-04-18T03:27:47Z

## Summary by CodeRabbit

- **New Features**
  - Comprehensive settings API (multiple sections) and collection endpoints (providers, MCP servers, environments, hooks)
  - Daemon restart flow with durable operations, status endpoints and CLI relaunch helper
  - Observability log-tail streaming via SSE
  - Extension management HTTP endpoints (list/install/enable/disable)
  - Config overlay persistence that preserves file structure/comments and workspace/global scopes
  - HTTP mutation guarding: remote mutations restricted to loopback hosts

- **Documentation**
  - OpenAPI schemas updated for settings and extension endpoints

## Walkthrough

Adds a complete settings subsystem: API contracts, conversion layer, HTTP/UDS handlers (including SSE log-tail), config overlay persistence with TOML AST edits, MCP JSON sidecar writes, daemon restart orchestration with persisted operations and relaunch helper, detached process utilities, extension handlers, and extensive tests across server, UDS, spec, persistence, and restart flows.

## Changes

| Cohort / File(s)                                                                                                                                                                                                                                                      | Summary                                                                                                                                                                                                                         |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Project scaffolding** <br> `\.compozy/tasks/settings-ui/qa/issues/.gitkeep`, `\.compozy/tasks/settings-ui/qa/screenshots/.gitkeep`                                                                                                                                  | Add empty .gitkeep files to preserve QA directories.                                                                                                                                                                            |
| **Deps** <br> `go.mod`                                                                                                                                                                                                                                                | Added indirect toml dependency `github.com/pelletier/go-toml v1.9.5`.                                                                                                                                                           |
| **API contract** <br> `internal/api/contract/settings.go`, `internal/api/contract/settings_test.go`                                                                                                                                                                   | New comprehensive settings API types, enums, request/response payloads, and JSON-shape tests for MutationResult.                                                                                                                |
| **API conversion** <br> `internal/api/core/conversions.go`                                                                                                                                                                                                            | New conversion helpers mapping internal settings envelopes to API payloads with validation and time/field normalization.                                                                                                        |
| **API errors & interfaces** <br> `internal/api/core/errors.go`, `internal/api/core/interfaces.go`                                                                                                                                                                     | Added settings sentinel errors + status mapping; added SettingsService and SettingsRestartController interfaces and SettingsRestartOperation type.                                                                              |
| **Core handlers** <br> `internal/api/core/handlers.go`, `internal/api/core/settings.go`, `internal/api/core/settings_*.go`                                                                                                                                            | Wired Settings and SettingsRestart into BaseHandlers; implemented full settings HTTP/SSE handlers (sections, collections, restart), plus extensive handler unit tests and internal tests.                                       |
| **HTTP API: routes, middleware, handlers** <br> `internal/api/httpapi/routes.go`, `internal/api/httpapi/middleware.go`, `internal/api/httpapi/extensions.go`, `internal/api/httpapi/handlers.go`, `internal/api/httpapi/server.go`, `internal/api/httpapi/*.go tests` | Registered settings and extensions routes, added loopback-only privileged-mutation guard, extension handlers/service interface, server wiring options, and tests enforcing loopback mutation restrictions and transport parity. |
| **UDS API wiring & tests** <br> `internal/api/udsapi/*.go`                                                                                                                                                                                                            | Wired settings/restart into UDS server, registered settings routes for UDS, and added UDS handler/tests and transport-parity integration tests.                                                                                 |
| **OpenAPI spec** <br> `internal/api/spec/spec.go`, `internal/api/spec/settings_test.go`                                                                                                                                                                               | Added settings enums/operations to OpenAPI document and tests validating operation presence, transports, and schemas.                                                                                                           |
| **Config persistence & MCP JSON** <br> `internal/config/persistence.go`, `internal/config/merge.go`, `internal/config/mcpjson.go`, `internal/config/mcpjson_write.go`, tests                                                                                          | Introduced TOML-overlay AST-aware editor, EditConfigOverlay, MCP sidecar JSON read/write (camelCase/snake_case support) and comprehensive persistence tests/integration tests.                                                  |
| **Bootstrap & home layout** <br> `internal/config/bootstrap.go`, `internal/config/home.go`, `internal/config/*_test.go`                                                                                                                                               | Refactored bootstrap to use overlay edits; added RestartsDir/HomePaths.RestartsDir and tests.                                                                                                                                   |
| **Daemon: restart & settings runtime** <br> `internal/daemon/restart.go`, `internal/daemon/settings.go`, `internal/daemon/boot.go`, `internal/daemon/daemon.go`, tests                                                                                                | Durable restart operation state machine with file-backed store, relaunch helper flow, settings runtime surface exposing status, restart controller, boot integration, and wide test coverage (unit + integration).              |
| **Daemon relaunch CLI & procutils** <br> `internal/cli/daemon.go`, `internal/cli/root.go`, `internal/cli/daemon_wait_test.go`, `internal/procutil/*.go`                                                                                                               | Added hidden `daemon relaunch` command, delegated detached spawning to procutil, and implemented cross-platform SpawnDetachedLoggedProcess with log capture and enriched error context.                                         |
| **Settings classification** <br> `internal/settings/classify.go`                                                                                                                                                                                                      | Added mutation classification logic to determine apply-now vs restart-required behavior.                                                                                                                                        |

## Sequence Diagram

```mermaid
sequenceDiagram
    participant Client
    participant APIHandler as API Handler
    participant SettingsService as Settings Service
    participant Persistence as Persistence Layer
    participant Disk as Disk/File

    Client->>APIHandler: GET /api/settings/general?scope=global
    APIHandler->>SettingsService: GetSection(ctx, req)
    SettingsService->>Persistence: Load merged config for scope
    Persistence->>Disk: Read files (global/workspace)
    Persistence-->>SettingsService: SectionEnvelope
    SettingsService-->>APIHandler: SectionEnvelope -> API payload
    APIHandler-->>Client: 200 OK JSON

    Client->>APIHandler: PATCH /api/settings/general (loopback)
    APIHandler->>SettingsService: UpdateSection(ctx, updateReq)
    SettingsService->>Persistence: EditConfigOverlay(scope, mutations)
    Persistence->>Disk: Read overlay
    Persistence->>Disk: Parse TOML AST, apply edits, write overlay
    Persistence->>SettingsService: Validated merged Config
    SettingsService-->>APIHandler: MutationResult (applied/restart_required)
    APIHandler-->>Client: 200/202 JSON
```

```mermaid
sequenceDiagram
    participant Client
    participant RestartAPI as Restart API
    participant RestartController as Restart Controller
    participant Daemon as Daemon
    participant RestartStore as Restart Store
    participant RelaunchHelper as Relaunch Helper
    participant NewDaemon as New Daemon
    participant Disk as Disk/File

    Client->>RestartAPI: POST /api/settings/actions/restart
    RestartAPI->>RestartController: RequestRestart(ctx)
    RestartController->>Daemon: RequestRestart()
    Daemon->>RestartStore: Create operation (pending -> stopping) and persist
    RestartStore->>Disk: Write JSON operation file
    Daemon->>Daemon: Send SIGTERM to old process
    Daemon->>RelaunchHelper: Spawn "daemon relaunch" (env: operation_id)
    RestartController-->>RestartAPI: RestartOperation (operation_id, status)
    RestartAPI-->>Client: 202 Accepted + status URL

    RelaunchHelper->>RestartStore: Poll operation status (waiting for stopping)
    RelaunchHelper->>Daemon: Poll release conditions (socket/file/lock)
    RelaunchHelper->>RestartStore: Transition waiting_release -> starting (persist)
    RelaunchHelper->>NewDaemon: Spawn replacement ("daemon start")
    NewDaemon->>RestartStore: markRestartReady(new_pid) (persist ready)
    RestartStore->>Disk: Update operation file to ready
    Client->>RestartAPI: GET /api/settings/actions/restart/{operation_id}
    RestartAPI->>RestartController: GetRestartOperation(ctx, id)
    RestartController->>RestartStore: Read operation file
    RestartController-->>RestartAPI: RestartOperation (ready, new_pid)
    RestartAPI-->>Client: 200 OK JSON
```
