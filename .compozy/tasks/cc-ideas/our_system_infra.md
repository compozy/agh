# Our System: Infrastructure, Tests & Supporting Code

## Current Architecture

### Overview

AGH (Agent Network Framework) is a Go-based kernel that orchestrates multiple autonomous coding agents (Claude Code, Codex, OpenCode, Pi) into collaborative workgroups. It is a single-binary, local-first system running as a daemon process. The binary is built from `cmd/agh/main.go` and wired through `internal/cli.NewRootCommand()` using Cobra.

The project module is `github.com/pedronauck/agh`, targeting Go 1.25.0.

### Package Layout

| Path                    | Responsibility                                                                                                                     | Maturity                               |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| `cmd/agh/`              | Main entry point (28 lines), delegates to `internal/cli`                                                                           | Stable                                 |
| `internal/config/`      | TOML config loading, validation, merge, roles, playbooks, home paths                                                               | Mature (8 files, ~1600 lines test)     |
| `internal/logger/`      | Structured logging with slog + charmbracelet/log human mode                                                                        | Stable (156 lines + 177 lines test)    |
| `internal/version/`     | Build metadata (3 vars, 1 function)                                                                                                | Stable (14 lines + 30 lines test)      |
| `internal/kernel/`      | Core kernel: boot, shutdown, sessions, registries, workgroups, hooks, resilience, API, skills, prompts, dashboard adapter, runtime | Mature (31 files, heavy test coverage) |
| `internal/cli/`         | Cobra command tree: 30+ subcommands, daemon client, output formatting                                                              | Mature (32 files, 15 test files)       |
| `internal/skills/`      | Skill loading, registry, catalog builder, verification, ClawHub client                                                             | Recently added (7 test files)          |
| `internal/prompt/`      | Prompt assembly: templates per agent type, context injection, role/playbook/skills catalog insertion                               | Mature (3 test files)                  |
| `internal/transport/`   | Embedded NATS broker, UDS bridge, scope validation                                                                                 | Stable                                 |
| `internal/state/`       | SQLite per-session state (WAL mode), event/blackboard/status tables                                                                | Stable                                 |
| `internal/pty/`         | PTY allocation via creack/pty, ring buffer, window sizing                                                                          | Stable                                 |
| `internal/registry/`    | In-memory registries (agent, workgroup, role catalog, driver)                                                                      | Stable                                 |
| `internal/dashboard/`   | Web dashboard server, WebSocket hub, Gin router, go:embed frontend                                                                 | Stable                                 |
| `internal/drivers/`     | Pluggable agent drivers (claude, codex, opencode, pi)                                                                              | Stable                                 |
| `internal/toon/`        | TOON (Token-Oriented Object Notation) renderer for LLM output                                                                      | Stable                                 |
| `internal/frontmatter/` | YAML frontmatter parser/formatter for Markdown files                                                                               | Stable                                 |
| `internal/roles/`       | Role management helpers                                                                                                            | Stable                                 |
| `internal/session/`     | Session helpers                                                                                                                    | Stable                                 |
| `internal/testutil/`    | Shared test fixtures (`catalogfiles` package)                                                                                      | Stable                                 |

### Build System

Uses `mage` (Go task runner) wrapped by a Makefile:

- `make verify` -- blocking gate: runs `fmt -> lint -> test -> build` serially
- `make fmt` -- gofmt
- `make lint` -- golangci-lint with zero tolerance
- `make test` -- `go test -race`
- `make build` -- compile binary
- `make deps` -- `go mod tidy && go mod verify`

---

## Logger (`internal/logger/`)

### What Exists

Two files: `logger.go` (156 lines) and `logger_test.go` (177 lines).

**Core design**: Factory function `New(level string, opts ...Option) (*slog.Logger, error)` that:

1. Parses log level string (debug/info/warn/error, case-insensitive, whitespace-trimmed)
2. Creates either a JSON slog handler (default) or a charmbracelet/log human-readable handler
3. Optionally wraps in an `observedHandler` decorator that clones records to a `LogObserver` interface

**Functional options pattern**:

- `WithWriter(io.Writer)` -- redirect output (tests use `bytes.Buffer`)
- `WithObserver(LogObserver)` -- receive cloned records for inspection
- `WithHumanHandler()` -- swap JSON for charmbracelet/log styled text

**Observer pattern**: The `LogObserver` interface (`Observe(context.Context, slog.Record)`) enables test-time record inspection without coupling to output format. The `observedHandler` wraps any `slog.Handler` and forwards cloned records.

**JSON handler customization**: Level keys are uppercased (`INFO`, `DEBUG`, etc.) via `ReplaceAttr`.

### Maturity

Stable. Clean functional options pattern, comprehensive table-driven tests covering all levels, JSON output format, observer wiring, level filtering, and human handler mode.

---

## Version (`internal/version/`)

### What Exists

Two files: `version.go` (14 lines) and `version_test.go` (30 lines).

Three package-level vars (`Version`, `Commit`, `Date`) with defaults (`"dev"`, `"none"`, `"unknown"`). Single `String()` function returns formatted version string.

### Maturity

Complete. Trivial package. Build metadata is injected via `-ldflags` at compile time.

---

## Config (`internal/config/`)

### What Exists

Eight source files totaling approximately 1200 lines of production code and 1600 lines of tests.

**File breakdown**:

- `config.go` -- Top-level types (`Config`, `LimitsConfig`, `RuntimeConfig`, `DashboardConfig`, `MetaConfig`, `Paths`, `HomePaths`, `Project`), defaults, validation, `Load()`, `LoadFromRoot()`, `Init()`, `ResolvePaths()`
- `limits.go` -- Duration parsing helpers, `LimitsConfig.Validate()`
- `roles.go` -- `RoleConfig` type (YAML frontmatter), `ArtifactStatus` (draft/approved), `BootstrapKind`, `LoadRoles()`, `FindRole()`, `ResolveRole()`, `ResolveBootstrap()`, draft versioning
- `playbooks.go` -- `Playbook` type (YAML frontmatter), `LoadPlaybooks()`, `FindPlaybook()`, validation
- `discovery.go` -- Save/approve/draft lifecycle for roles and playbooks, artifact candidate collection, version management
- `merge.go` -- Two-layer config merge (global + workspace), overlay pattern with pointer fields for selective override, role and playbook merge with workspace-wins-on-collision
- `home.go` -- Global AGH home directory resolution (`$AGH_HOME` or `~/.agh/`), `HomePaths`, `EnsureHomeLayout()`
- `config_test.go` -- 1600 lines of comprehensive tests

**Key patterns**:

1. **TOML config with overlay merge**: Config uses `BurntSushi/toml`. The merge system uses a parallel `configOverlay` struct with pointer fields (`*int`, `*string`, `*bool`) so only explicitly set values override defaults. Global config loads first, workspace config overlays on top.

2. **Roles as Markdown with YAML frontmatter**: Roles are `.md` files parsed via `internal/frontmatter`. The frontmatter contains `name`, `description`, `type`, `driver`, `model`. The body is the `SystemPrompt`. Draft versioning uses filename conventions: `name.draft_v1.md`, `name.draft_v2.md`.

3. **Playbooks as Markdown with YAML frontmatter**: Same pattern as roles. Frontmatter has `name`, `description`, `domain`, `tags`. Body is the playbook content.

4. **Draft-to-approve lifecycle**: Meta-learning system where agents can create draft roles/playbooks during sessions. Drafts are versioned (`draft_v1`, `draft_v2`, ...). `ApproveRole()` / `ApprovePlaybook()` promotes the highest draft to approved status. Auto-approve mode skips the draft step.

5. **Workspace-wins merge**: When both global (`~/.agh/`) and workspace (`.agh/`) provide a role/playbook with the same name, workspace takes precedence. Same for config keys.

6. **Known driver whitelist**: Only `claude`, `codex`, `opencode`, `pi` are valid driver names. OpenCode requires a `mode` field (`tui` or `server`).

7. **Strict validation**: Unknown TOML keys are rejected. Ports must be 1-65535. Durations must parse and be positive. Role types must be one of `master/worker/advisor/reviewer/researcher`.

**Default configuration**:

- 5 max sessions, 50 agents/session, 3 max workgroup depth, 10 agents/workgroup, 100 total agents
- 3 restart attempts, 5s backoff base, 30s health check interval, 60s readiness timeout
- Dashboard on `localhost:2123`, ring buffer 1MB, terminal 120x36
- Default driver: `claude`, supervisor model: `opus`, advisor model: `sonnet`
- Meta-learning disabled by default

### Maturity

Highly mature. Extensive test coverage including round-trip load tests, merge precedence tests, draft lifecycle tests, validation edge cases, idempotent init tests, and home path resolution tests.

---

## Kernel (`internal/kernel/`)

### What Exists

31 files covering the core runtime. Key components:

**Core types** (`types.go`, ~976 lines):

- `AgentDriver` interface -- 8 methods (Name, Start, SendMessage, Stop, BuildHookConfig, ParseHookEvent, HealthCheck, DetectReady)
- `StartOpts` -- agent launch configuration with tools, terminal size, env vars
- `Session` -- isolated execution context with its own registries, state store, PTY manager, suture supervisor, WsHub, NATS subscriptions, circuit breakers, health checker
- `SessionManager` -- manages all sessions with concurrent-safe maps, pending reservations, agent counting
- `Kernel` -- global shared infrastructure (Config, NATS, UDS, HTTP, registries, skills, prompt templates, daemon lock)
- `UnimplementedDriver` -- placeholder for future drivers
- Session state machine: `starting -> active -> stopping -> stopped`

**Boot sequence** (`kernel.go`):
13-step boot: acquire lock -> write daemon info -> load config -> init logger -> start NATS -> start UDS -> load roles -> init drivers -> load skills -> load prompts -> init sessions -> start HTTP -> start signal handler

**Session lifecycle** (`session_manager.go`):

- `Create()` -- generates ID, captures workspace, creates session dir, merges config, loads roles, opens SQLite, initializes all per-session subsystems, spawns supervisor + advisor bootstrap agents
- `Stop()` -- transitions to stopping, calls shutdown function, closes WsHub, unsubscribes NATS, stops driver processes, closes SQLite, deregisters
- `Resume()` -- restores stopped session from metadata, re-opens store, re-creates registries, re-spawns bootstrap agents with historical context
- Concurrent safety via `sync.RWMutex` and pending session tracking

**API layer** (`api.go`, `api_lifecycle.go`, `api_messaging_state.go`):

- HTTP handlers for session management, agent messaging, state operations
- UDS bridge handles routing to correct session based on request context

**Skills integration** (`skills.go`):

- Four-level skill loading hierarchy (bundled -> user home -> .agents -> workspace)
- Skills are frozen after kernel boot
- Workspace-scoped snapshot for session-specific skill catalogs

**Prompt catalog** (`prompt_catalog.go`):

- Renders role and playbook catalogs as prompt sections for master agents
- Non-master agents do not receive role/playbook catalogs

### Test Coverage (16 test files)

The kernel test suite is extensive:

- `kernel_test.go` -- Boot sequence, subsystem initialization, config loading, skill registry freezing, daemon lock, signal handling, shutdown ordering
- `session_manager_test.go` -- Full session lifecycle: create/stop/resume, limits enforcement, concurrent create/stop, cleanup on failure, workspace capture, config override, role merging, skills catalog injection, tool assignment by agent type, terminal size propagation
- `api_lifecycle_test.go` -- HTTP API for session start/stop/list
- `api_messaging_state_test.go` -- Messaging and state API tests
- `skills_test.go` -- Four-level skill loading with precedence
- `session_registries_test.go`, `session_config_test.go` -- Per-session registry and config tests
- `workgroups_test.go`, `workgroups_internal_test.go` -- Workgroup management tests
- `hooks_test.go` -- Hook system tests
- `resilience_test.go` -- Error handling and circuit breaker tests
- `runtime_test.go` -- Agent runtime tests
- `types_test.go` -- Type validation and state machine tests
- `dashboard_adapter_test.go`, `dashboard_integration_test.go` -- Dashboard integration tests

### Maturity

Mature. The kernel is the most complex package with comprehensive integration tests. All 26 M1 tasks are completed.

---

## CLI (`internal/cli/`)

### What Exists

32 files implementing 30+ Cobra subcommands:

**Command tree** (from `root.go`):
`agh` -> version, start, status, stop, session, workgroup, topology, spawn, kill, ps, whoami, attach, dashboard, roles, playbooks, send, broadcast, escalate, state, context, agent-status, events, wait, done, skill, hook-event, install

**Key subsystems**:

- `daemon.go` -- Daemon client over UDS, session context resolution
- `output.go` -- Dual output mode (human styled via lipgloss, TOON for agents). Auto-detection: flag > `COLLAB_AGENT` env var > default human
- `skill.go` -- `agh skill list/view/search/install/remove` commands
- `install.go` -- `agh install` for workspace initialization
- `session.go` -- Session start/stop/list/resume commands
- `lifecycle.go` -- Agent lifecycle (done, wait)
- `messaging.go` -- send, broadcast, escalate
- `state.go` -- State read/append operations
- `workgroup.go` -- Workgroup create/list/topology
- `roles.go` / `playbooks.go` -- Role/playbook management including draft approve
- `hooks.go` -- Hook event forwarding from agent processes

### Test Coverage (15 test files)

Tests use mock daemon clients and verify command execution, output formatting, and error handling.

### Maturity

Mature. Full command coverage with both human and TOON output modes.

---

## Skills (`internal/skills/`)

### What Exists

Recently added package (7 source files + 7 test files):

- `types.go` -- `SkillMeta`, `Skill`, `SkillEntry`, `SkillSource` (Bundled/UserHome/UserAgents/ProjectAgents/Workspace), `SkillSnapshot`
- `loader.go` -- Parse `SKILL.md` files (YAML frontmatter + Markdown body)
- `registry.go` -- Skill registry with multi-source loading, freeze capability, snapshot generation with filtering
- `catalog.go` -- XML catalog builder for system prompt injection (`<available_skills>` XML block)
- `eligibility.go` -- OS-based filtering, disabled list support
- `verify.go` -- Security scanning for prompt injection patterns (blocks skills containing suspicious phrases like "system prompt override")
- `clawhub.go` -- HTTP client for ClawHub marketplace (search with retry/backoff, download with tar.gz/zip/raw markdown support, path traversal protection)

**Four-level loading hierarchy** (lower overrides higher):

1. Bundled (go:embed) -- ships with the binary
2. User home (`~/.agh/skills/` + `~/.agents/skills/`)
3. Project agents (`.agents/skills/` in workspace)
4. Workspace (`.agh/skills/` in workspace)

**Security**: Downloaded skills are verified for prompt injection patterns before installation. Critical warnings cause installation to be blocked and cleaned up.

### Maturity

Recently completed (all 4 tasks done). Well-tested with comprehensive ClawHub client tests including retry logic, archive format handling, and security verification.

---

## Prompt System (`internal/prompt/`)

### What Exists

Prompt assembler composes agent system prompts from three layers:

1. **Template** -- Per agent type (`master`, `worker`, `advisor`, `reviewer`, `researcher`). Built-in templates define behavior sections: ROLE, COMMANDS AVAILABLE, RULES, EXAMPLES, ERROR HANDLING. Master gets additional sections: MUST NOT, MUST, BASH RESTRICTIONS, AGENT CAPABILITIES, DELEGATION PROTOCOL, WRITING AGENT PROMPTS, CONTINUE VS SPAWN, SELF-IMPROVEMENT, BOOT SEQUENCE.

2. **Specialization** -- Per role, from role's `SystemPrompt` field. Inserted between template and context.

3. **Context** -- Per session: Goal, Domain, AgentID, WorkgroupID, WorkgroupName, AgentType, RoleName.

**Additional injections**:

- Skills catalog (XML `<available_skills>` block) -- inserted between specialization and context
- Roles catalog -- only for master agents
- Playbooks catalog -- only for master agents
- Additional sections (workspace path, historical context for resumed sessions)

**Ordering**: Template -> Specialization -> Skills -> Roles Catalog -> Playbooks Catalog -> Context -> Additional Sections

### Maturity

Mature. Comprehensive tests verify section ordering, optional section omission, role/type mismatch rejection, and catalog injection.

---

## Testing Patterns

### Approach

The codebase consistently follows these testing patterns:

1. **Table-driven tests with subtests**: `t.Run()` with parallel execution is the universal pattern. Example from `logger_test.go`:

   ```go
   testCases := []struct {
       name    string
       level   string
       wantErr bool
   }{...}
   for _, tc := range testCases {
       tc := tc
       t.Run(tc.name, func(t *testing.T) {
           t.Parallel()
           ...
       })
   }
   ```

2. **`t.Parallel()`**: Used extensively for independent subtests. Both at the top-level test function and within subtests.

3. **`t.TempDir()`**: Used instead of manual temp directory management for filesystem isolation.

4. **`t.Helper()`**: All test helper functions are marked. Examples: `writeFile()`, `mustResolvePaths()`, `assertExists()`, `findRoleByStatus()`, `bootTestKernel()`.

5. **`t.Cleanup()`**: Used for teardown (kernel shutdown, directory cleanup, working directory restoration).

6. **`t.Setenv()`**: Used for environment variable overrides in tests (e.g., `AGH_HOME`).

7. **Interface-based mocking**: Test doubles implement production interfaces. Key examples:
   - `sessionTestDriver` implements `AgentDriver` with recording capabilities (tracks starts, stops, sent messages)
   - `staticAgentRegistry` implements `AgentRegistryStore` for lightweight test scenarios
   - `testObserver` implements `LogObserver` for log inspection
   - `roundTripFunc` implements `http.RoundTripper` for HTTP client testing

8. **No external test frameworks**: Pure stdlib testing. No testify, no gomock. Assertions are written inline with `t.Fatalf()` / `t.Errorf()`.

9. **Shared test fixtures**: `internal/testutil/catalogfiles` package provides `RoleMarkdown()` and `PlaybookMarkdown()` helpers to generate canonical test data.

10. **Integration tests**: Several packages have integration test files (e.g., `daemon_integration_test.go`, `runtime_integration_test.go`, `dashboard_integration_test.go`) that test real subsystem interactions.

11. **Concurrency tests**: The session manager has explicit concurrent create/stop tests that verify thread safety. Idempotent shutdown tests verify concurrent callers.

12. **httptest.NewServer**: ClawHub client tests use `httptest.NewServer` to simulate the marketplace API with various response scenarios (retry, error, different archive formats).

### Test Infrastructure

- `bootTestKernel(t)` -- Creates a fully initialized kernel with mock driver, ephemeral home directory, and auto-cleanup
- `bootSessionTestKernel(t, home, cfg, driver)` -- Same but with explicit config and driver
- `newKernelHome(t)` -- Creates ephemeral home paths under `/tmp/aghk-*`
- `newDashboardListener(t)` -- Allocates a random TCP port for dashboard testing
- `writeSkillFixture(t, root, dir, name, description)` -- Creates a SKILL.md fixture in the correct directory structure
- `assertContainsPromptFragments(t, value, ...fragments)` -- Verifies prompt assembly output

---

## Project Status

### Completed Task Sets

**spec-v2 (26 tasks -- ALL COMPLETED)**:
The entire M1 milestone is done. All 26 tasks from project scaffolding through to the Pi driver are completed:

- Foundation: scaffolding, config, SQLite state, NATS transport, registry, PTY/ring buffer
- Drivers: Claude Code, Codex, OpenCode, Pi
- Kernel: multi-session rework, prompt assembler, TOON renderer, hook system, workgroups, resilience, boot/shutdown orchestration, session manager
- CLI: daemon, session, messaging/state, workgroups/discovery, lifecycle/hooks
- Meta-learning, web dashboard (server + frontend)

**skills-system (4 tasks -- ALL COMPLETED)**:

- Task 01: Skills package core (types, loader, verify, eligibility)
- Task 02: Registry, catalog builder, and bundled skills
- Task 03: Kernel integration (boot, spawn, prompt)
- Task 04: CLI commands and ClawHub client

### In-Progress Work (Uncommitted Changes)

Based on `git status`, there are significant uncommitted changes across:

**Modified files** (staged or working tree):

- `internal/cli/daemon.go`, `daemon_integration_test.go`, `daemon_test.go` -- Daemon client changes
- `internal/cli/dashboard.go`, `install.go`, `lifecycle.go`, `messaging.go`, `playbooks.go`, `roles.go`, `root.go`, `runtime.go`, `session.go`, `state.go`, `workgroup.go` -- Broad CLI changes (likely the dual output mode work)
- `internal/kernel/api.go`, `api_lifecycle_test.go`, `kernel.go`, `kernel_test.go`, `session_manager.go`, `session_manager_test.go`, `types.go` -- Kernel changes
- `internal/prompt/assembler.go`, `assembler_test.go` -- Prompt changes

**New files** (untracked):

- `internal/cli/human/` -- Human-friendly output formatters (new package)
- `internal/cli/output.go`, `output_test.go` -- Dual output mode implementation
- `internal/cli/skill.go`, `skill_test.go` -- Skill CLI commands
- `internal/kernel/skills.go`, `skills_test.go` -- Kernel skill integration
- `internal/skills/clawhub.go`, `clawhub_test.go` -- ClawHub marketplace client

### Design Documents & Plans

**Active designs** (`docs/plans/`):

- `2026-03-31-skills-system-design.md` -- Skills system design (implemented)
- `2026-03-31-cli-dual-output-design.md` -- Dual output mode: human (lipgloss) + TOON, auto-detection via env var
- `2026-03-31-supervisor-orchestration-enforcement-design.md` -- Tool filtering by agent type, MUST NOT constraints for master, bash restricted to `agh` commands for master

**Task directories for in-progress features**:

- `.compozy/tasks/cli-dual-output/` -- Has techspec + 3 ADRs
- `.compozy/tasks/supervisor-orchestration/` -- Has techspec + 4 ADRs

**Historical designs** (`docs/plans/`):

- Architecture improvements, agent network framework (v1 and v2), agent driver spec, dashboard design, multi-session design, research findings, canvas library research

**Spec v2 documentation** (`docs/spec-v2/`):
16 specification documents covering executive summary, architecture, kernel, agents, workgroups, drivers, CLI, configuration, data models, resilience, meta-learning, testing, development sequence, observability, risks/decisions, examples, and web dashboard.

### Review Issues

`.compozy/tasks/spec-v2/reviews-001/` contains 100+ review issue files (issue_001.md through issue_162.md) from a systematic code review round.

### Roles & Playbooks

The project does not have a `.agh/` directory in the workspace root (not initialized as an AGH workspace itself). Role and playbook definitions are loaded at runtime from:

- Global: `~/.agh/roles/*.md` and `~/.agh/playbooks/*.md`
- Workspace: `{cwd}/.agh/roles/*.md` and `{cwd}/.agh/playbooks/*.md`

Role types supported: `master`, `worker`, `advisor`, `reviewer`, `researcher`

---

## Key Patterns

### Configuration Patterns

1. **TOML with overlay merge**: Pointer-based overlay structs ensure only explicitly set values override defaults
2. **Two-layer resolution**: Global (`~/.agh/`) + Workspace (`.agh/`) with workspace-wins semantics
3. **Strict validation**: Unknown keys rejected, known driver whitelist, positive duration enforcement
4. **Defaults as code**: `config.Default()` provides the canonical built-in configuration

### Logging Patterns

1. **`log/slog` throughout**: Standard library structured logging
2. **JSON for production, charmbracelet/log for humans**: Switchable via `WithHumanHandler()`
3. **Observer decorator**: Test-time record inspection without output coupling
4. **Functional options**: Clean, extensible constructor pattern

### Error Handling Patterns

1. **Wrapped errors**: `fmt.Errorf("context: %w", err)` throughout
2. **Sentinel errors**: `ErrNotImplemented`, `ErrSessionNotFound`, `ErrSessionExists`, `ErrMetaLearningDisabled`, `ErrRegistryFrozen`
3. **Custom error types**: `sessionNotFoundError`, `sessionAlreadyExistsError` with `Unwrap()` for `errors.Is()` matching
4. **Validation chains**: Switch statements that return the first validation failure

### Concurrency Patterns

1. **`sync.RWMutex`**: Read-heavy registries (sessions, agents, workgroups)
2. **`sync.Once`**: Shutdown idempotency (`shutdownOnce`, `stopOnce`)
3. **`sync.WaitGroup`**: Parallel session shutdown
4. **Channel-per-agent**: Message serialization to prevent interleaving
5. **Context cancellation**: Lifecycle contexts for sessions with explicit cancel
6. **`atomic.Uint32`**: Lock-free lifecycle state tracking

### Testing Patterns

1. **Table-driven + parallel**: Universal approach
2. **Interface mocking**: No reflection-based mocks; hand-written test doubles
3. **Full kernel boot in tests**: `bootTestKernel()` creates real NATS, UDS, HTTP, registries
4. **Ephemeral filesystem**: `t.TempDir()` + `t.Setenv("AGH_HOME", ...)` for isolation
5. **Recording test doubles**: `sessionTestDriver` records all Start/Stop/SendMessage calls for assertion
6. **No external test libraries**: Pure `testing` package

### ID Generation

- `xid` library for collision-free, time-sortable 20-char IDs
- Prefixed: `ag-` (agents), `wg-` (workgroups), `ev-` (events)
- Sessions use raw xid without prefix

### Prompt Assembly

- Three-layer composition: Template (agent type behavior) + Specialization (role domain) + Context (session)
- Skills catalog injected as XML between specialization and context
- Role/playbook catalogs only for master agents
- Strict ordering with section existence checks
