# TechSpec: Supervisor Orchestration Enforcement

## Executive Summary

The AGH master/supervisor agent executes code directly instead of delegating to worker agents. This happens because `StartOpts.Tools` is never populated during agent spawn (giving all agents unrestricted tool access) and the master template lacks explicit behavioral constraints.

This techspec defines a dual enforcement system: runtime tool filtering via `StartOpts.Tools` per agent type, combined with a rewritten master template that includes MUST NOT constraints, a delegation protocol, agent capabilities table, and roles/playbooks catalog injection.

Key architectural decisions: dual runtime + prompt enforcement (ADR-001), frontmatter catalog injection for roles/playbooks (ADR-002), canonical tool sets per agent type (ADR-003), and master template rewrite with coordinator pattern (ADR-004).

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      Agent Spawn Path                           │
│                                                                  │
│  bootstrapAgent() / spawnAgent()                                │
│  │                                                               │
│  ├─ 1. Resolve agent type (master/worker/advisor/reviewer/...)  │
│  │                                                               │
│  ├─ 2. ToolsForAgentType(type) ──────────────────────────────┐  │
│  │      master:     [read, grep, glob, list, bash]           │  │
│  │      worker:     [read, write, edit, bash, grep, glob, list] │
│  │      advisor:    [read, grep, glob, list]                 │  │
│  │      reviewer:   [read, bash, grep, glob, list]           │  │
│  │      researcher: [read, grep, glob, list]                 │  │
│  │                                                               │
│  ├─ 3. StartOpts.Tools = toolList                               │
│  │                                                               │
│  ├─ 4. BuildRoleCatalog() + BuildPlaybookCatalog()  (master only)│
│  │                                                               │
│  ├─ 5. prompt.Assemble(opts) ──────────────────────────────────┐│
│  │      Template (master.md — REWRITTEN)                       ││
│  │      + Role Specialization                                  ││
│  │      + Skills Catalog                                       ││
│  │      + Roles Catalog (NEW — frontmatter)                    ││
│  │      + Playbooks Catalog (NEW — frontmatter)                ││
│  │      + Context                                              ││
│  │                                                              ││
│  ├─ 6. driver.Start(ctx, startOpts)                             │
│  │      Driver translates Tools to driver-specific format       │
│  │      Claude: --allowedTools    Pi: --tools                   │
│  │      OpenCode: config map      Codex: sandbox mode           │
│  │                                                               │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. At agent spawn, `ToolsForAgentType()` returns the canonical tool set for the agent type
2. The tool set is written into `StartOpts.Tools`
3. For master agents, roles/playbooks catalogs are rendered from the session's `RoleCatalog` and workspace playbooks
4. `prompt.Assemble()` composes the system prompt with all sections including catalogs
5. The driver translates canonical tool names to driver-specific names and enforces the restriction
6. The agent starts with both prompt constraints (MUST NOT) and runtime tool restrictions (filtered tools)

## Implementation Design

### 1. Tool Sets per Agent Type

New file `internal/prompt/tools.go`:

```go
// ToolsForAgentType returns the canonical kernel tool names
// allowed for the given agent type.
func ToolsForAgentType(agentType string) ([]string, error) {
    normalized := strings.ToLower(strings.TrimSpace(agentType))
    tools, ok := agentTypeTools[normalized]
    if !ok {
        return nil, fmt.Errorf("prompt: unknown agent type %q for tool resolution", agentType)
    }
    return append([]string(nil), tools...), nil
}
```

Tool sets defined as package-level map:

| Agent Type | Tools | Rationale |
|---|---|---|
| `master` | `read`, `grep`, `glob`, `list`, `bash` | Read context + `agh` CLI via bash (restricted by prompt) |
| `worker` | `read`, `write`, `edit`, `bash`, `grep`, `glob`, `list` | Full implementation capability |
| `advisor` | `read`, `grep`, `glob`, `list` | Read-only consultation, communication via `agh` CLI subset |
| `reviewer` | `read`, `bash`, `grep`, `glob`, `list` | Read + verification commands (tests, lint, build) |
| `researcher` | `read`, `grep`, `glob`, `list` | Strictly read-only, no bash |

### 2. Roles/Playbooks Catalog Rendering

New file `internal/prompt/role_catalog.go`:

```go
// BuildRoleCatalog renders a text catalog of role frontmatter
// for injection into the master agent's system prompt.
func BuildRoleCatalog(roles []*config.RoleConfig) string

// BuildPlaybookCatalog renders a text catalog of playbook
// frontmatter for injection into the master agent's system prompt.
func BuildPlaybookCatalog(playbooks []config.Playbook) string
```

**Roles catalog output format:**
```
AVAILABLE ROLES:
The following roles can be assigned when spawning agents. Use `agh roles get <name>` to read the full definition.

- planner (type: master) — Decomposes goals into execution plans
- executor (type: worker) — Disciplined task execution with verification
- code-reviewer (type: reviewer) — Evidence-driven code review
- architect (type: advisor) — Strategic architecture analysis
```

**Playbooks catalog output format:**
```
AVAILABLE PLAYBOOKS:
The following playbooks define coordination strategies. Use `agh playbooks get <name>` to read the full content.

- feature-development (domain: engineering) — End-to-end feature implementation workflow
- bug-investigation (domain: engineering) — Structured debugging and fix verification
```

Each entry renders the YAML frontmatter fields: name, type (for roles), domain (for playbooks), and description. Tags are omitted from the catalog for brevity. Empty catalogs return an empty string (no section injected).

### 3. Prompt Assembly Changes

Modified `internal/prompt/assembler.go`:

```go
type AssembleOptions struct {
    Type               string
    Template           string
    RoleName           string
    Role               *config.RoleConfig
    SkillsCatalog      string
    RolesCatalog       string    // NEW
    PlaybooksCatalog   string    // NEW
    Context            Context
    AdditionalSections []string
}
```

The `Assemble()` function appends non-empty `RolesCatalog` and `PlaybooksCatalog` as sections between `SkillsCatalog` and `Context`, following the existing pattern for skills catalog injection.

### 4. Master Template Rewrite

Complete rewrite of `internal/prompt/templates/master.md` with the following sections:

**Section 1 — ROLE:**
Identity as coordinator. "You coordinate your workgroup — you do NOT implement." Clear statement that every implementation, verification, and research task must be delegated to specialized agents.

**Section 2 — MUST NOT:**
```
You MUST NOT:
1. Write, edit, or create files directly
2. Run build, test, or lint commands directly
3. Execute implementation code
4. Install dependencies or modify package files
5. Modify git state (commit, push, branch, merge)
6. Use Bash for anything other than `agh` commands
```

**Section 3 — MUST:**
```
You MUST:
1. Delegate all implementation to worker agents
2. Delegate all verification to reviewer agents
3. Consult advisor agents for strategic decisions
4. Use researcher agents for information gathering
5. Synthesize findings before delegating follow-up work
6. Verify completion with concrete evidence from agents
```

**Section 4 — BASH RESTRICTIONS:**
Explicit constraint that Bash is available ONLY for `agh` commands. Prohibits file operations, code execution, package management, git operations via bash.

**Section 5 — COMMANDS AVAILABLE:**
Existing `agh` CLI command reference (discovery, runtime, messaging, workgroups, state, lifecycle). Kept from current template.

**Section 6 — AGENT CAPABILITIES:**
Table showing which tools each agent type receives when spawned. This teaches the master what each type can do, enabling informed delegation decisions.

**Section 7 — DELEGATION PROTOCOL:**
Four-phase workflow adapted from Claude Code coordinator pattern:
1. RESEARCH — Spawn researchers to understand the problem (parallel when independent)
2. SYNTHESIZE — Master reads findings, identifies approach (master does the thinking)
3. IMPLEMENT — Spawn workers with specific, synthesized specs (file paths, line numbers, exact changes)
4. VERIFY — Spawn reviewers to prove code works (tests, typechecks, lint; be skeptical)

Anti-pattern: "Never write 'based on your findings' — that delegates understanding to the worker."

**Section 8 — WRITING AGENT PROMPTS:**
Rules for effective delegation adapted from Claude Code:
- Workers cannot see your conversation — every prompt must be self-contained
- Include file paths, line numbers, error messages, exact changes expected
- State what "done" looks like
- For research: "Report findings — do not modify files"
- For implementation: "Run tests, then report completion with evidence"

**Section 9 — CONTINUE VS SPAWN:**
Decision framework:
- High context overlap → continue existing agent
- Different concern or polluting context → spawn fresh
- Correcting failure → continue (agent has error context)
- Verifying another agent's work → spawn fresh (fresh eyes)

**Section 10 — SELF-IMPROVEMENT:**
```
If no existing role fits a need, create a new one:
  agh roles create --name <name> --type <type>
Then use `agh roles approve <name>` when validated.
```

**Section 11 — BOOT SEQUENCE:**
Kept from current template with minor adjustments. On boot:
1. Review available roles and playbooks (now visible in prompt)
2. Based on the goal and available playbooks, plan approach
3. Create first workgroup with `agh workgroup create`
4. Spawn agents and begin coordination

**Section 12 — RULES:**
Kept from current template: minimal staffing preference, hook handling, delegation boundaries, verify before closing.

**Section 13 — ERROR HANDLING:**
Kept from current template: inspect context and events, unblock or replace stalled agents, gather state before acting.

### 5. Kernel Integration — `bootstrapAgent()`

In `internal/kernel/session_manager.go`, the `bootstrapAgent()` function is modified:

**Tool population** (after line 393, before line 399):
```go
tools, err := prompt.ToolsForAgentType(agentType)
if err != nil {
    _ = session.WorkgroupManager.KillAgent(ctx, registered.ID)
    return nil, err
}
startOpts.Tools = tools
```

**Catalog injection in `buildBootstrapPrompt()`** (lines 427-467):
For master type agents, render roles and playbooks catalogs and pass to `prompt.Assemble()`:

```go
var rolesCatalog, playbooksCatalog string
if agent != nil && strings.EqualFold(strings.TrimSpace(agent.Type), "master") {
    if session.RoleCatalog != nil {
        rolesCatalog = prompt.BuildRoleCatalog(session.RoleCatalog.List())
    }
    playbooks, err := config.LoadPlaybooks(playbooksDir)
    if err == nil {
        playbooksCatalog = prompt.BuildPlaybookCatalog(playbooks)
    }
}
```

These are passed to `prompt.Assemble()` via the new `RolesCatalog` and `PlaybooksCatalog` fields.

### 6. Kernel Integration — `spawnAgent()`

In `internal/kernel/api.go`, the `spawnAgent()` function is modified:

**Tool population** (after line 474, where StartOpts is built):
```go
tools, err := prompt.ToolsForAgentType(registered.Type)
if err != nil {
    _ = session.WorkgroupManager.KillAgent(ctx, registered.ID)
    return SessionAgentResponse{}, err
}
startOpts.Tools = tools
```

**Catalog injection** follows the same pattern as `buildBootstrapPrompt()` — for master type agents, render roles/playbooks catalogs into the prompt assembly options.

### 7. Playbook Loading for Catalog

Playbooks are loaded from the session workspace for catalog rendering. The loading uses the existing `config.LoadPlaybooks()` function with the workspace playbooks directory. No new `PlaybookCatalogStore` interface is needed — playbooks are loaded on demand during prompt assembly for master agents only.

The playbooks directory is resolved via `config.ResolvePaths(workspace)` which returns `paths.PlaybooksDir`.

## Testing Strategy

### Unit Tests

**`internal/prompt/tools_test.go`:**
- `TestToolsForAgentType` — table-driven: each type returns expected tool set
- `TestToolsForAgentTypeUnknown` — unknown type returns error
- `TestToolsForAgentTypeReturnsDefensiveCopy` — modifying returned slice doesn't affect source

**`internal/prompt/role_catalog_test.go`:**
- `TestBuildRoleCatalog` — renders correct format with name, type, description
- `TestBuildRoleCatalogEmpty` — nil/empty input returns empty string
- `TestBuildRoleCatalogSorting` — entries sorted by name
- `TestBuildPlaybookCatalog` — renders correct format with name, domain, description
- `TestBuildPlaybookCatalogEmpty` — nil/empty input returns empty string

**`internal/prompt/assembler_test.go`:**
- Extend existing tests to verify `RolesCatalog` and `PlaybooksCatalog` sections are included when non-empty
- Verify sections are omitted when empty strings

### Integration Tests

**`internal/kernel/session_manager_test.go`:**
- Verify `bootstrapAgent()` populates `StartOpts.Tools` for supervisor (master type)
- Verify `bootstrapAgent()` populates `StartOpts.Tools` for advisor
- Verify master agent's system prompt contains roles catalog section
- Verify master agent's system prompt contains playbooks catalog section
- Verify non-master agents do NOT receive roles/playbooks catalogs

**`internal/kernel/api_test.go`:**
- Verify `spawnAgent()` populates `StartOpts.Tools` based on resolved agent type

## Files to Modify

| File | Change | Complexity |
|------|--------|-----------|
| `internal/prompt/tools.go` (new) | `ToolsForAgentType()` function + canonical tool sets | Low |
| `internal/prompt/tools_test.go` (new) | Unit tests for tool sets | Low |
| `internal/prompt/role_catalog.go` (new) | `BuildRoleCatalog()`, `BuildPlaybookCatalog()` | Low |
| `internal/prompt/role_catalog_test.go` (new) | Unit tests for catalog rendering | Low |
| `internal/prompt/assembler.go` | Add `RolesCatalog`, `PlaybooksCatalog` to `AssembleOptions` | Low |
| `internal/prompt/assembler_test.go` | Extend tests for new catalog fields | Low |
| `internal/prompt/templates/master.md` | Complete rewrite with 13 sections | Medium |
| `internal/kernel/session_manager.go` | Populate `StartOpts.Tools` + catalogs in `bootstrapAgent()` | Medium |
| `internal/kernel/api.go` | Populate `StartOpts.Tools` in `spawnAgent()` | Low |
| `internal/kernel/session_manager_test.go` | Integration tests for tool population + catalogs | Medium |
| `internal/kernel/api_test.go` | Integration tests for spawn tool population | Low |

## Files That Do NOT Change

| File | Reason |
|------|--------|
| Driver implementations (claude, codex, opencode, pi) | Already support `StartOpts.Tools` translation |
| `internal/prompt/templates/worker.md` | Already well-defined |
| `internal/prompt/templates/advisor.md` | Already well-defined |
| `internal/prompt/templates/reviewer.md` | Already well-defined |
| `internal/prompt/templates/researcher.md` | Already well-defined |
| `internal/skills/*` | Unchanged |
| `internal/config/roles.go` | `RoleConfig` struct already has needed fields |
| `internal/config/playbooks.go` | `Playbook` struct already has needed fields |
| CLI commands | Unchanged |

## Architecture Decision Records

- [ADR-001: Runtime + Prompt Dual Enforcement](adrs/adr-001.md) — Both runtime tool filtering and prompt constraints, neither alone sufficient
- [ADR-002: Frontmatter Catalog Injection](adrs/adr-002.md) — Roles/playbooks frontmatter in prompt, full content via CLI (same pattern as skills)
- [ADR-003: Canonical Tool Sets per Agent Type](adrs/adr-003.md) — Fixed tool sets per type, populated into StartOpts.Tools at spawn time
- [ADR-004: Master Template Rewrite](adrs/adr-004.md) — 13-section template with MUST NOT, delegation protocol, capabilities table

## Implementation Sequencing

```
Phase 1 (no dependencies):
├── prompt/tools.go + tests           — ToolsForAgentType()
├── prompt/role_catalog.go + tests    — BuildRoleCatalog(), BuildPlaybookCatalog()
└── prompt/templates/master.md        — Template rewrite

Phase 2 (depends on Phase 1):
├── prompt/assembler.go + tests       — New AssembleOptions fields
└── kernel/api.go + tests             — StartOpts.Tools in spawnAgent()

Phase 3 (depends on Phase 1 + 2):
└── kernel/session_manager.go + tests — StartOpts.Tools + catalogs in bootstrapAgent()
```

Phase 1 tasks are fully independent and can be implemented in parallel. Phase 2 depends on the new types from Phase 1. Phase 3 ties everything together in the bootstrap path.
