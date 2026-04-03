# TechSpec: CLI Dual Output Mode

## Executive Summary

AGH adds dual output mode to its CLI: `human` (styled with Charmbracelet lipgloss) and `toon` (current structured format). Format is resolved automatically via environment variable detection — agents get `toon` because the kernel sets `COLLAB_AGENT`/`AGI_AGENT` on every spawned process; humans get styled output by default. An explicit `-o`/`--output` flag overrides detection. The logger swaps its slog handler to `charmbracelet/log` in human mode for colored, readable log output.

Key decisions: env var detection over TTY detection (ADR-001), Charmbracelet stack (ADR-002), incremental migration with toon fallback (ADR-003).

## System Architecture

### Component Overview

```
┌──────────────────────────────────────────────────────────────┐
│                     internal/cli/                            │
│                                                              │
│  ┌─────────────┐    ┌──────────────┐    ┌────────────────┐  │
│  │  root.go    │    │  output.go   │    │  command files  │  │
│  │  registers  │───▶│  resolves    │◀───│  call           │  │
│  │  -o flag    │    │  format      │    │  writeOutput()  │  │
│  └─────────────┘    └──────┬───────┘    └────────────────┘  │
│                            │                                 │
│              ┌─────────────┴─────────────┐                   │
│              │                           │                   │
│     ┌────────▼────────┐        ┌─────────▼───────┐          │
│     │ internal/cli/   │        │  internal/toon/  │          │
│     │ human/          │        │  (unchanged)     │          │
│     │ lipgloss tables │        │  TOON format     │          │
│     │ styled output   │        │                  │          │
│     └─────────────────┘        └──────────────────┘          │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐     │
│  │  internal/logger/                                   │     │
│  │  slog.Handler swap: JSON (agent) ↔ charm/log (human)│     │
│  └─────────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────────┘
```

**Data flow:**

1. Root command registers `-o`/`--output` as persistent flag (inherits to all subcommands)
2. Each command calls `writeOutput(cmd, humanFn, toonFn)` instead of `writeCommandOutput(cmd, rendered)`
3. `writeOutput` calls `resolveOutputFormat(cmd)` which checks: explicit flag → env var → default
4. Dispatches to the appropriate renderer; falls back to toon if human renderer is nil

## Implementation Design

### Core Interfaces

```go
// internal/cli/output.go

// OutputFormat represents the CLI output format.
type OutputFormat string

const (
    OutputHuman OutputFormat = "human"
    OutputToon  OutputFormat = "toon"
)

// resolveOutputFormat determines the output format from flag,
// env var, or default.
func resolveOutputFormat(cmd *cobra.Command) OutputFormat

// RenderFunc produces a rendered string or an error.
type RenderFunc func() (string, error)

// writeOutput dispatches to humanFn or toonFn based on format.
// If humanFn is nil (command not yet migrated), falls back to toon.
func writeOutput(cmd *cobra.Command, humanFn, toonFn RenderFunc) error
```

```go
// internal/cli/human/styles.go

// Shared lipgloss styles for consistent visual language.
var (
    HeaderStyle  lipgloss.Style // Bold, primary color
    LabelStyle   lipgloss.Style // Dim/gray for field labels
    ValueStyle   lipgloss.Style // Default foreground
    SuccessStyle lipgloss.Style // Green
    ErrorStyle   lipgloss.Style // Red
    WarnStyle    lipgloss.Style // Yellow
    InfoStyle    lipgloss.Style // Cyan
    DimStyle     lipgloss.Style // Subdued text
)

// Status indicator constants
const (
    IconSuccess = "✓"
    IconError   = "✗"
    IconPending = "⟳"
    IconBullet  = "•"
)
```

```go
// internal/cli/human/renderer.go

// Human renderers mirror toon.Render* signatures.
func RenderAgents(agents []toon.AgentView) (string, error)
func RenderVerboseAgents(agents []toon.VerboseAgentView) (string, error)
func RenderAgentIdentity(view toon.AgentIdentityView) (string, error)
func RenderAgentRuntime(view toon.AgentRuntimeView) (string, error)
func RenderKernelStatus(view toon.KernelStatusView) (string, error)
func RenderSessionSummary(view toon.SessionSummaryView) (string, error)
func RenderSessionList(items []toon.SessionSummaryView) (string, error)
func RenderSessionDetail(view toon.SessionDetailView) (string, error)
func RenderMessage(view toon.MessageView) (string, error)
func RenderStatus(entry state.StatusEntry) (string, error)
func RenderBlackboard(entries []state.BlackboardEntry) (string, error)
func RenderEvents(entries []state.EventEntry) (string, error)
func RenderWorkgroups(rows []toon.WorkgroupRecord) (string, error)
func RenderTopologyTree(tree toon.TopologyTreeView) (string, error)
func RenderRoleList(items []toon.RoleSummaryView) (string, error)
func RenderRoleDetail(view toon.RoleDetailView) (string, error)
func RenderPlaybookList(items []toon.PlaybookSummaryView) (string, error)
func RenderPlaybookDetail(view toon.PlaybookDetailView) (string, error)
func RenderDashboard(view toon.DashboardView) (string, error)
func RenderContext(view toon.ContextView) (string, error)
```

```go
// internal/logger/logger.go (modified)

// WithHumanHandler configures charmbracelet/log as the slog handler.
func WithHumanHandler() Option
```

### Data Models

No new data models. The human renderers consume the same view types already defined in `internal/toon/` (`AgentView`, `SessionSummaryView`, `KernelStatusView`, etc.). This is intentional — the view layer is format-agnostic.

### API Endpoints

No API changes. This is purely a CLI presentation layer change.

## Integration Points

### Root Command Flag Registration

- **Integration point**: `internal/cli/root.go` `newRootCommand()`
- **Change**: Add `cmd.PersistentFlags().StringP("output", "o", "", "Output format: human, toon")`
- **Behavior**: Persistent flag inherits to all subcommands

### Command Migration Pattern

Each CLI command file changes from:

```go
// Before
rendered, err := toon.RenderAgents(rows)
if err != nil { return err }
return writeCommandOutput(cmd, rendered)
```

To:

```go
// After
return writeOutput(cmd,
    func() (string, error) { return human.RenderAgents(rows) },
    func() (string, error) { return toon.RenderAgents(rows) },
)
```

### Logger Handler Swap

- **Integration point**: `internal/logger/logger.go` `New()`
- **Change**: New `WithHumanHandler()` option that uses `charmbracelet/log` as the underlying `slog.Handler`
- **Behavior**: When enabled, log output becomes colored and human-readable while all `slog.*` call sites remain unchanged
- **Caller**: `internal/cli/daemon.go` `newStartCommand` — resolve format before creating logger, pass option when human

### Kernel Environment Variables (existing, no change)

- `internal/kernel/api.go:821-858` `agentRuntimeEnv()` already sets `COLLAB_AGENT`/`AGI_AGENT`
- `resolveOutputFormat()` reads these env vars to detect agent context
- No kernel changes required

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/cli/output.go` | New | Format resolution and output dispatcher. No risk — new isolated file. | Create from scratch |
| `internal/cli/human/` | New | Human renderers package (~5 files). No risk — new package. | Create from scratch |
| `internal/cli/root.go` | Modified | Add persistent `-o` flag. Minimal risk — additive 1-line change. | Add PersistentFlags registration |
| `internal/cli/daemon.go` | Modified | Migrate commands to `writeOutput`. Low risk — mechanical refactor. | Replace `writeCommandOutput` calls |
| `internal/cli/session.go` | Modified | Same migration pattern. Low risk. | Replace `writeCommandOutput` calls |
| `internal/cli/runtime.go` | Modified | Same migration pattern. Low risk. | Replace `writeCommandOutput` calls |
| `internal/cli/state.go` | Modified | Same migration pattern. Low risk. | Replace `writeCommandOutput` calls |
| `internal/cli/messaging.go` | Modified | Same migration pattern. Low risk. | Replace `writeCommandOutput` calls |
| `internal/cli/workgroup.go` | Modified | Same migration pattern. Low risk. | Replace `writeCommandOutput` calls |
| `internal/cli/lifecycle.go` | Modified | Same migration pattern. Low risk. | Replace `writeCommandOutput` calls |
| `internal/cli/roles.go` | Modified | Same migration pattern. Low risk. | Replace `writeCommandOutput` calls |
| `internal/cli/playbooks.go` | Modified | Same migration pattern. Low risk. | Replace `writeCommandOutput` calls |
| `internal/cli/dashboard.go` | Modified | Same migration pattern. Low risk. | Replace `writeCommandOutput` calls |
| `internal/cli/skill.go` | Modified | Same migration pattern. Low risk. | Replace `writeCommandOutput` calls |
| `internal/cli/install.go` | Modified | Same migration pattern. Low risk. | Replace `writeCommandOutput` calls |
| `internal/logger/logger.go` | Modified | Add `WithHumanHandler` option. Low risk — additive change, existing behavior unchanged. | Add new option and charm/log handler |
| `go.mod` | Modified | Add 2 direct deps. Low risk. | `go get` lipgloss and charm/log |
| `internal/toon/` | No change | Remains the agent renderer. Zero risk. | None |
| `internal/kernel/` | No change | Env vars already set. Zero risk. | None |
| Prompt templates | No change | Agents auto-detect via env var. Zero risk. | None |

## Testing Approach

### Unit Tests

**`internal/cli/output_test.go`**:
- `resolveOutputFormat` returns `human` when no flag and no env var
- `resolveOutputFormat` returns `toon` when `COLLAB_AGENT` is set
- `resolveOutputFormat` returns `toon` when `AGI_AGENT` is set
- `resolveOutputFormat` returns explicit flag value regardless of env var
- `writeOutput` dispatches to humanFn when format is human
- `writeOutput` dispatches to toonFn when format is toon
- `writeOutput` falls back to toonFn when humanFn is nil and format is human

**`internal/cli/human/*_test.go`**:
- Each renderer produces non-empty output for valid input
- Output contains expected data values (names, IDs, states)
- Output does not contain TOON format markers
- Empty input produces clean empty/no-data output
- Table renderers handle single and multiple rows

**`internal/logger/logger_test.go`** (extend existing):
- `WithHumanHandler` produces non-JSON output
- `WithHumanHandler` output contains log level keywords (INFO, ERROR, etc.)
- Default behavior unchanged when `WithHumanHandler` not used

Mock requirements:
- `t.Setenv()` for env var tests (no manual cleanup needed)
- `bytes.Buffer` as cmd.SetOut() for output capture
- No external mocks — all rendering is pure functions

### Integration Tests

- End-to-end: command with `-o human` produces styled output, `-o toon` produces TOON
- Env var: command with `COLLAB_AGENT` set produces TOON without flag
- Fallback: unmigrated command produces TOON even when format is human

## Development Sequencing

### Build Order

1. **`go.mod` dependencies** — `go get github.com/charmbracelet/lipgloss@latest` and `go get github.com/charmbracelet/log@latest` (no dependencies)
2. **`internal/cli/output.go`** — format resolution + `writeOutput` dispatcher (depends on cobra only)
3. **`internal/cli/root.go`** — register persistent `-o` flag (depends on step 2 for format constants)
4. **`internal/cli/human/styles.go`** — shared lipgloss styles and icons (depends on lipgloss)
5. **`internal/cli/human/tables.go`** — generic table builder helper using lipgloss/table (depends on step 4)
6. **`internal/cli/human/renderer.go`** — high-traffic renderers: `RenderKernelStatus`, `RenderAgents`, `RenderSessionList`, `RenderSessionDetail` (depends on steps 4-5)
7. **`internal/logger/logger.go`** — add `WithHumanHandler` option (depends on charm/log)
8. **Migrate Phase 2 commands** — `daemon.go` (status/start/stop), `runtime.go` (ps/spawn/kill), `session.go` (depends on steps 2, 6)
9. **`internal/cli/human/renderer_extra.go`** — remaining renderers for messaging, state, workgroup, roles, playbooks, etc. (depends on steps 4-5)
10. **Migrate Phase 3 commands** — all remaining CLI files (depends on steps 2, 9)

### Technical Dependencies

- **`github.com/charmbracelet/lipgloss`**: Terminal styling and table rendering. Install via `go get`.
- **`github.com/charmbracelet/log`**: Human-friendly slog handler. Install via `go get`.
- No other new dependencies. `mattn/go-isatty` already in dep tree (indirect via gin).

## Monitoring and Observability

No new observability. The output format is a presentation concern that doesn't affect kernel behavior, metrics, or operational logging. The logger handler swap only changes formatting, not content or destinations.

| Event | Level | Structured Fields |
|-------|-------|-------------------|
| Output format resolved | Debug | `format`, `source` (flag/env/default) |

## Technical Considerations

### Key Decisions

See Architecture Decision Records below for full details.

- **Env var detection** (ADR-001): Deterministic, no TTY edge cases, leverages existing kernel behavior.
- **Charmbracelet stack** (ADR-002): Industry standard for Go CLI styling, slog-compatible logger.
- **Incremental migration** (ADR-003): Nil-fallback pattern allows command-by-command adoption.

### Known Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Lipgloss output breaks in exotic terminals | Low | Lipgloss auto-degrades color profiles. `NO_COLOR` env var disables all styling. |
| Human renderers diverge from toon view types | Medium | Human renderers import and consume the same view types from `internal/toon/`. No parallel type definitions. |
| Migration stalls, some commands stay toon-only | Medium | Prioritize top 5 human-used commands. Toon fallback means no broken UX. |
| charm/log handler missing slog features | Low | Wrap with existing `observedHandler` pattern. Observer hooks still work. |

## Architecture Decision Records

- [ADR-001: Environment Variable Detection](adrs/adr-001.md) — Use COLLAB_AGENT/AGI_AGENT env vars to auto-default agents to toon format
- [ADR-002: Charmbracelet Stack](adrs/adr-002.md) — Use lipgloss for styling and charmbracelet/log as human-friendly slog handler
- [ADR-003: Incremental Migration with Toon Fallback](adrs/adr-003.md) — Nil-fallback dispatcher enables command-by-command migration
