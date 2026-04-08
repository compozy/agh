# TechSpec: Skills System v2 — MCP Bridge, Lifecycle Hooks, and Marketplace

## Executive Summary

This TechSpec extends the existing skills system (Increment 1) with three capabilities: **MCP lazy-load** (skills declare MCP server dependencies in frontmatter, daemon provisions them at session start), **lifecycle hooks** (skills react to session events via subprocess commands), and **marketplace** (pluggable registry interface with ClawHub as default backend, hash-based provenance).

The primary trade-off: we add runtime complexity (subprocess execution for hooks, HTTP client for marketplace, MCP consent tracking) in exchange for a self-improving agent system where skills can declare tool dependencies, react to session lifecycle, and be discovered/installed from external registries.

All three features extend the existing `internal/skills/` package and integrate through established patterns: `StartOpts.MCPServers` for MCP bridging, `session.Notifier` for lifecycle events, and Cobra subcommands for CLI. No new packages beyond `internal/skills/marketplace/` and its `clawhub/` sub-package.

Reality check against the current codebase: today `internal/skills/` exposes `Skill`, `SkillMeta`, `SkillSource`, `Registry`, `CatalogProvider`, `VerifyContent`, and `Watcher`. The MCP, hook, provenance, and marketplace symbols below are planned additions, not existing runtime types yet.

Key decisions: trust-tiered MCP consent (ADR-001), hybrid hook execution model (ADR-002), pluggable registry interface (ADR-003), hash-based provenance without cryptographic signing (ADR-004).

---

## System Architecture

### Component Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                         AGH Daemon                               │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                   internal/skills/                         │  │
│  │                                                            │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │  │
│  │  │ Registry │  │  Loader  │  │  Verify  │  │ Catalog  │  │  │
│  │  │(existing)│  │(extended)│  │(existing)│  │(existing)│  │  │
│  │  └────┬─────┘  └────┬─────┘  └──────────┘  └──────────┘  │  │
│  │       │              │                                     │  │
│  │  ┌────▼──────────────▼─────────────────────────────────┐  │  │
│  │  │              NEW: MCP Resolver                      │  │  │
│  │  │  (collects MCPServerDecl from active skills,       │  │  │
│  │  │   applies trust tiers, injects into StartOpts)     │  │  │
│  │  └─────────────────────┬───────────────────────────────┘  │  │
│  │                        │                                   │  │
│  │  ┌─────────────────────▼───────────────────────────────┐  │  │
│  │  │              NEW: HookRunner                        │  │  │
│  │  │  (dispatches subprocess hooks on session events,   │  │  │
│  │  │   JSON stdin/stdout, timeout, fail-open)           │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  │                                                            │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │           NEW: marketplace/ package                 │  │  │
│  │  │  ┌──────────────┐  ┌─────────────────────────────┐ │  │  │
│  │  │  │   Registry   │  │  clawhub/ (implementation)  │ │  │  │
│  │  │  │  (interface)  │  │  Search / Download / Info   │ │  │  │
│  │  │  └──────────────┘  └─────────────────────────────┘ │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────┐  ┌────────────────┐  ┌──────────────────────┐  │
│  │ session/     │  │ daemon/        │  │ cli/                 │  │
│  │ Notifier     │◄─┤ notifierFanout │  │ skill search/install │  │
│  │ (existing)   │  │ + hook phase   │  │ skill remove/update  │  │
│  └──────────────┘  └────────────────┘  └──────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### Data Flow — MCP Lazy-Load

1. Loader parses `metadata.agh.mcp_servers` from SKILL.md frontmatter into `[]MCPServerDecl`
2. At session creation, `MCPResolver` collects MCPServerDecl from all active skills for the workspace
3. Trust tier check: bundled/user/additional/workspace auto-approved; marketplace requires consent
4. Approved servers are merged with agent/provider MCP servers via existing `MergeMCPServers()`
5. Combined list passed to `StartOpts.MCPServers` → ACP driver spawns them alongside agent

### Data Flow — Lifecycle Hooks

1. Loader parses `metadata.agh.hooks` from SKILL.md frontmatter into `[]HookDecl`
2. `notifierFanout` grows a dedicated post-notifier hook phase so ordinary `session.Notifier` callbacks still run first
3. On `OnSessionCreated` / `OnSessionStopped`, the hook phase resolves the session workspace and collects hooks from active skills
4. Subprocess hooks execute in source precedence order (bundled → marketplace → user → additional → workspace) with JSON stdin, stdout captured
5. Hook output (enrichment data) is logged but not yet injected into session context (future: `on_prompt_assembly`)

### Data Flow — Marketplace

1. User runs `agh skill search "query"` → CLI dispatches to `marketplace.Registry` interface
2. `clawhub.Client` calls ClawHub API, returns `[]SkillListing`
3. `agh skill install @author/name` → downloads skill archive → extracts to `~/.agh/skills/`
4. Computes SHA-256 hash, runs `VerifyContent`, stores metadata in `.agh-meta.json` sidecar
5. Skill enters a running daemon registry on the next watcher poll (or after restart). Immediate refresh would require a daemon API/RPC that is out of scope here.

---

## Implementation Design

### Core Interfaces

**Extended Skill types** — added to `internal/skills/types.go`:

```go
// MCPServerDecl declares an MCP server dependency in skill frontmatter.
type MCPServerDecl struct {
    Name    string            `yaml:"name"`
    Command string            `yaml:"command"`
    Args    []string          `yaml:"args,omitempty"`
    Env     map[string]string `yaml:"env,omitempty"`
}

// HookDecl declares a lifecycle hook in skill frontmatter.
type HookDecl struct {
    Event   HookEvent         `yaml:"event"`
    Command string            `yaml:"command"`
    Args    []string          `yaml:"args,omitempty"`
    Timeout time.Duration     `yaml:"timeout,omitempty"`
    Env     map[string]string `yaml:"env,omitempty"`
}

// HookEvent identifies when a hook fires.
type HookEvent string
const (
    HookSessionCreated HookEvent = "on_session_created"
    HookSessionStopped HookEvent = "on_session_stopped"
)
```

**Marketplace registry interface** — new `internal/skills/marketplace/registry.go`:

```go
// Registry defines the marketplace backend contract.
type Registry interface {
    Search(ctx context.Context, query string, opts SearchOpts) ([]SkillListing, error)
    Download(ctx context.Context, slug string) (*SkillArchive, error)
    Info(ctx context.Context, slug string) (*SkillDetail, error)
}
```

**HookRunner** — new `internal/skills/hooks.go`:

```go
// HookRunner dispatches subprocess hooks for skill lifecycle events.
type HookRunner struct {
    logger *slog.Logger
}

// RunHooks executes all hooks matching the given event, in precedence order.
func (hr *HookRunner) RunHooks(ctx context.Context, event HookEvent,
    skills []*Skill, payload HookPayload) []HookResult
```

**MCPResolver** — new `internal/skills/mcp.go`:

```go
// MCPResolver collects and resolves MCP server declarations from skills.
type MCPResolver struct {
    allowedMarketplace []string // from config: approved marketplace consent entries/slugs
    logger             *slog.Logger
}

// Resolve returns MCP servers from active skills, applying trust tiers.
func (mr *MCPResolver) Resolve(skills []*Skill) []aghconfig.MCPServer
```

### Data Models

**Skill struct extension** — fields added to existing `Skill`:

```go
type Skill struct {
    // ... existing fields ...
    MCPServers    []MCPServerDecl  // Parsed from metadata.agh.mcp_servers
    Hooks         []HookDecl       // Parsed from metadata.agh.hooks
    Provenance    *Provenance      // Marketplace: hash + source metadata
    InstalledFrom string           // Marketplace: registry slug (e.g., "@author/name")
}
```

**SkillSource extension** — add marketplace source:

```go
const (
    SourceBundled     SkillSource = iota
    SourceMarketplace                     // NEW: ~/.agh/skills/ installed from registry
    SourceUser
    SourceAdditional
    SourceWorkspace
)
```

Note: `SourceMarketplace` is inserted between `SourceBundled` and `SourceUser` to maintain precedence order (marketplace overrides bundled, user overrides marketplace).
Manual skills in `~/.agh/skills/` without a `.agh-meta.json` sidecar remain `SourceUser`; only sidecar-backed installs become `SourceMarketplace`.

**Provenance** — marketplace metadata sidecar:

```go
type Provenance struct {
    Hash      string    `json:"hash"`       // SHA-256 of SKILL.md at install time
    Registry  string    `json:"registry"`   // e.g., "clawhub"
    Slug      string    `json:"slug"`       // e.g., "@author/skill-name"
    Version   string    `json:"version"`
    InstalledAt time.Time `json:"installed_at"`
}
```

Stored as `.agh-meta.json` in the skill directory. Loaded by the registry during skill parsing.

**Marketplace types** — in `internal/skills/marketplace/`:

```go
type SkillListing struct {
    Slug        string `json:"slug"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Author      string `json:"author"`
    Version     string `json:"version"`
    Downloads   int    `json:"downloads"`
}

type SkillArchive struct {
    Slug    string
    Version string
    Data    io.Reader  // tar.gz stream
}

type SkillDetail struct {
    SkillListing
    Readme      string   `json:"readme"`
    MCPServers  []string `json:"mcp_servers,omitempty"`
    Tags        []string `json:"tags,omitempty"`
}

type SearchOpts struct {
    Limit  int
    Offset int
}
```

**HookPayload / HookResult** — JSON protocol for subprocess hooks:

```go
type HookPayload struct {
    SessionID string `json:"session_id"`
    AgentName string `json:"agent_name"`
    Workspace string `json:"workspace"`
    Event     string `json:"event"`
}

type HookResult struct {
    SkillName string
    Event     HookEvent
    Output    string        // stdout from subprocess
    Error     error         // nil on success
    Duration  time.Duration
}
```

**Config extensions** — added to `SkillsConfig`:

```go
type SkillsConfig struct {
    // ... existing fields ...
    AllowedMarketplaceMCP []string `toml:"allowed_marketplace_mcp,omitempty"`
    Marketplace           MarketplaceConfig `toml:"marketplace,omitempty"`
}

type MarketplaceConfig struct {
    Registry string `toml:"registry"` // default: "clawhub"
    BaseURL  string `toml:"base_url,omitempty"` // override for testing
}
```

This requires matching overlay and validation updates in the existing config pipeline; adding fields only to `internal/config/config.go` is not sufficient because the current runtime also relies on `internal/config/merge.go`.

### API Endpoints

No new HTTP API endpoints. Skills marketplace is managed exclusively via CLI. The existing workspace detail endpoint already includes skills in its response.
Current code note: `internal/cli/skill.go` currently exposes `list`, `view`, `info`, and `create`; the commands below are planned additions.

**New CLI Commands** (registered under `agh skill`):

| Command | Args | Description |
|---------|------|-------------|
| `agh skill search <query>` | `[--limit N]` | Search marketplace for skills |
| `agh skill install <slug>` | — | Install from marketplace to `~/.agh/skills/` |
| `agh skill remove <name>` | — | Delete installed skill directory |
| `agh skill update [name]` | `[--all]` | Update marketplace skills to latest version |

---

## Integration Points

### 1. Loader (internal/skills/loader.go) — MODIFIED

**Change**: After parsing YAML frontmatter, extract `metadata.agh.mcp_servers` and `metadata.agh.hooks` from the free-form `Metadata map[string]any` field into typed `[]MCPServerDecl` and `[]HookDecl` on the `Skill` struct.

**Implementation**: Add `parseAGHMetadata(skill *Skill)` function called after `parseSkillContent()`. Uses type assertions on `skill.Meta.Metadata["agh"]` to extract the nested maps. No new YAML parsing — the existing frontmatter parser already captures `metadata` as `map[string]any`.

### 2. Registry (internal/skills/registry.go) — MODIFIED

**Changes**:
- Add `SourceMarketplace` handling in `loadGlobalSkills()` — scan `~/.agh/skills/` for skills with `.agh-meta.json` sidecar, tag as `SourceMarketplace`
- Load provenance from `.agh-meta.json` during skill parsing
- On load: recompute SHA-256 hash of SKILL.md, compare with stored hash. Mismatch → re-scan + warning
- If marketplace skills must remain visible after a critical `VerifyContent` finding, the registry needs an explicit retained quarantine state. The current `processSkill()` path drops critically flagged skills entirely, and `Enabled = false` alone is insufficient because current catalog assembly does not filter disabled skills.

### 3. Session Manager (internal/session/manager_lifecycle.go) — MODIFIED

**Change**: Before constructing `StartOpts`, combine `resolved.MCPServers` with skill-declared MCP servers resolved from the active workspace skill set.

**Integration point**: `Manager.Create()` / `Resume()` do not currently call `registry.ForWorkspace()` directly. Active skill lookup happens in the prompt-assembly path (`skills.CatalogProvider.PromptSection()` via `Registry.ForWorkspace()`), so MCP resolution needs its own injected dependency on the session manager path rather than assuming registry access already exists in `manager_lifecycle.go`.

### 4. Daemon Notifier (internal/daemon/notifier.go) — MODIFIED

**Change**: Extend `notifierFanout` with a dedicated post-notifier hook dispatcher instead of registering hooks as an ordinary `session.Notifier`. This preserves ADR-002's required ordering of daemon-native notifiers first, subprocess hooks second.

**Execution**: The hook dispatcher needs both the skills registry and a way to recover the session's resolved workspace. `session.Session` currently carries `WorkspaceID` / `Workspace` strings, not a full `workspace.ResolvedWorkspace`, so the daemon must also pass a workspace resolver (or retain the resolved workspace on session creation) before it can call `Registry.ForWorkspace()`.

### 5. Daemon Boot (internal/daemon/boot.go) — MODIFIED

**Changes**:
- Construct `MCPResolver` plus a new session-manager dependency capable of loading active workspace skills during create/resume
- Construct `HookRunner` and wire the post-notifier hook dispatcher with access to both the skills registry and the workspace resolver
- Do not construct marketplace clients in daemon boot. Marketplace HTTP I/O is CLI-owned unless a new daemon API is introduced

### 6. CLI (internal/cli/skill.go) — MODIFIED

**Changes**: Add `search`, `install`, `remove`, `update` subcommands. These commands construct a marketplace client from config and dispatch through the `marketplace.Registry` interface.

### 7. ClawHub Client (internal/skills/marketplace/clawhub/) — NEW

**Purpose**: HTTP client for ClawHub API.

**Endpoints**:
- `GET /api/v1/skills?q=<query>&limit=<n>` → search
- `GET /api/v1/skills/<slug>` → info
- `GET /api/v1/skills/<slug>/download` → tar.gz archive

**Resilience**: Exponential backoff (1s initial, 30s max, 3 retries). Context-aware cancellation. HTTP timeout 30s.

---

## Impact Analysis

| Component | Impact | Description | Risk |
|-----------|--------|-------------|------|
| `internal/skills/types.go` | Modified | Add MCPServerDecl, HookDecl, Provenance, SourceMarketplace, HookEvent types | Low — additive |
| `internal/skills/loader.go` | Modified | Parse `metadata.agh` fields into typed structs | Low — existing Metadata map already captured |
| `internal/skills/registry.go` | Modified | Marketplace source handling, provenance loading, hash verification | Medium — changes to processSkill flow |
| `internal/skills/mcp.go` | New | MCPResolver: collect + filter MCP servers from skills | Low — isolated |
| `internal/skills/hooks.go` | New | HookRunner: subprocess dispatch with timeout | Medium — subprocess execution |
| `internal/skills/marketplace/` | New | Registry interface + types | Low — isolated package |
| `internal/skills/marketplace/clawhub/` | New | ClawHub HTTP client | Low — isolated, well-tested with httptest |
| `internal/session/{manager.go,manager_lifecycle.go}` | Modified | Inject an active-skills dependency and merge skill MCP servers into `StartOpts` | Medium — new manager dependency |
| `internal/daemon/{daemon.go,boot.go}` | Modified | Wire MCPResolver plus hook dependencies through the composition root; daemon boot does not own CLI marketplace clients | Medium — composition changes |
| `internal/daemon/notifier.go` | Modified | Add an explicit post-notifier hook phase instead of plain notifier registration | Medium — ordering semantics |
| `internal/cli/skill.go` | Modified | Add marketplace subcommands | Low — additive |
| `internal/config/{config.go,merge.go}` | Modified | Add marketplace consent config, overlay plumbing, and validation | Low — additive |

---

## Testing Approach

### Unit Tests

- **Loader** (`loader_test.go`): Parse `metadata.agh.mcp_servers` and `metadata.agh.hooks` from frontmatter. Verify MCPServerDecl and HookDecl populated correctly. Test missing/malformed AGH metadata gracefully ignored.
- **MCPResolver** (`mcp_test.go`): Trust tier filtering — bundled/user/additional/workspace auto-approved, marketplace blocked without consent. Consent allowlist. Duplicate server name dedup. Empty MCP list.
- **HookRunner** (`hooks_test.go`): Subprocess execution with JSON stdin. Timeout enforcement. Fail-open on hook error. Precedence ordering (bundled before marketplace before user before additional before workspace, alphabetical within level). Hook output capture.
- **Registry** (`registry_test.go`): Marketplace source detection via `.agh-meta.json`. Hash verification on load. Hash mismatch triggers re-scan. Confirm explicit quarantine behavior if retained state is introduced; otherwise confirm block-on-load semantics stay consistent. Provenance round-trip.
- **Config** (`config_test.go`, `merge_test.go`): Parse `skills.marketplace` and `allowed_marketplace_mcp`, validate defaults, and ensure workspace/global overlays merge correctly.
- **Marketplace Registry** (`marketplace/registry_test.go`): Interface compliance. SearchOpts pagination.
- **ClawHub Client** (`marketplace/clawhub/client_test.go`): Search response parsing. Download extraction. Retry logic with httptest server. Context cancellation. Error responses.
- **Provenance** (`provenance_test.go`): Hash computation. Sidecar write/read round-trip. Tamper detection.

### Integration Tests

- **MCP bridge** (`//go:build integration`): Create session with a skill declaring MCP server. Verify MCP server appears in StartOpts passed to driver.
- **Lifecycle hooks**: Create session with skill declaring `on_session_created` hook. Verify subprocess executed with correct JSON payload. Stop session, verify `on_session_stopped` hook fired.
- **Marketplace install**: Install skill from mock HTTP server. Verify `.agh-meta.json` is written with the correct hash and a running daemon observes the new skill after the watcher polls (or after restart if no daemon is running).

### Test Data

- Fixture SKILL.md files with `metadata.agh.mcp_servers` and `metadata.agh.hooks` in `testdata/`
- Mock ClawHub server using `httptest.NewServer`
- Simple hook scripts (shell) that echo JSON for subprocess testing

---

## Development Sequencing

### Build Order

1. **Types extension** (`types.go`) — Add MCPServerDecl, HookDecl, HookEvent, Provenance, SourceMarketplace. No dependencies.

2. **Loader extension** (`loader.go`) — Parse `metadata.agh` into typed fields on Skill. Depends on step 1.

3. **MCPResolver** (`mcp.go`) — Collect MCP servers from skills, apply trust tiers. Depends on step 1.

4. **Session manager integration** — Inject a new active-skills dependency and merge MCPResolver output into StartOpts. Depends on step 3.

5. **HookRunner** (`hooks.go`) — Subprocess dispatch with timeout and fail-open. Depends on step 1.

6. **Hook dispatch phase** (`daemon/notifier.go` modification) — Add a post-notifier hook phase that runs after ordinary notifiers. Depends on step 5.

7. **Provenance** (`provenance.go`) — Hash computation, sidecar read/write, tamper detection. Depends on step 1.

8. **Registry extension** (`registry.go`) — Marketplace source handling, provenance loading, hash check on load, and quarantine/block-on-load behavior. Depends on steps 2, 7.

9. **Marketplace interface** (`marketplace/registry.go`) — Registry interface + types. No dependencies beyond stdlib.

10. **ClawHub client** (`marketplace/clawhub/client.go`) — HTTP implementation of Registry interface. Depends on step 9.

11. **Marketplace CLI** (`cli/skill.go`) — search/install/remove/update commands. Depends on steps 8, 10, 13.

12. **Daemon boot wiring** — Construct MCPResolver, wire the post-notifier hook phase, and pass required dependencies into the session manager. Depends on steps 3 and 6.

13. **Config extension** (`config/config.go`, `config/merge.go`) — AllowedMarketplaceMCP, MarketplaceConfig fields, merge plumbing, and validation. Can be done in parallel with any step.

### Dependency Graph

```
1 (types) ──┬── 2 (loader) ──── 8 (registry ext) ──── 11 (marketplace CLI)
             │                        ▲
             ├── 3 (MCP resolver) ── 4 (session mgr) ──── 12 (boot wiring)
             │                                              ▲
             ├── 5 (hook runner) ──── 6 (hook phase) ──────┘
             │                                              │
             └── 7 (provenance) ──── 8 (registry ext)      │
                                                            │
             9 (marketplace iface) ── 10 (clawhub) ────────┘
                                                            │
             13 (config) ──────────────── 11 (cli) ────────┘
```

---

## Monitoring and Observability

All logging via `log/slog` per project conventions:

| Event | Level | Structured Fields |
|-------|-------|-------------------|
| Skill MCP server resolved | Info | `skill_name`, `mcp_server`, `source` |
| MCP server blocked (no consent) | Warn | `skill_name`, `mcp_server`, `source` |
| Hook dispatched | Debug | `skill_name`, `event`, `command` |
| Hook completed | Info | `skill_name`, `event`, `duration_ms` |
| Hook failed (timeout/error) | Warn | `skill_name`, `event`, `error`, `duration_ms` |
| Marketplace search | Debug | `query`, `result_count`, `registry` |
| Marketplace install | Info | `slug`, `version`, `hash`, `target_dir` |
| Marketplace install failed | Error | `slug`, `error` |
| Provenance hash mismatch | Warn | `skill_name`, `expected_hash`, `actual_hash` |
| Skill quarantined or blocked | Warn | `skill_name`, `reason` |

---

## Technical Considerations

### Key Decisions

- **Trust-tiered MCP consent** (ADR-001): Bundled/user/additional/workspace auto-approve; marketplace requires explicit consent stored in config.
- **Hybrid hook execution** (ADR-002): Built-in behaviors as Notifier callbacks, custom hooks as subprocesses. `on_prompt_assembly` deferred.
- **Pluggable marketplace** (ADR-003): `Registry` interface with ClawHub default. Future registries implement the same interface.
- **Hash-based provenance** (ADR-004): SHA-256 hash + VerifyContent scan. No cryptographic signing yet.

### Known Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Hook subprocess hangs beyond timeout | Medium | `context.WithTimeout` + process kill. Fail-open ensures session continues. |
| ClawHub API unavailable | Low | Graceful degradation — marketplace commands fail with clear error. All local functionality works offline. |
| Skill MCP server has harmful side effects | Low | Marketplace consent gate + VerifyContent scanning. User/additional/workspace skills are trusted by design. |
| Hash verification false positives on filesystem changes | Low | Hash only checked for marketplace skills. User-edited marketplace skills trigger expected warning. |
| Subprocess hooks on Windows | Medium | Use `exec.CommandContext` which handles cross-platform process management. Test on CI. |

### Deferred to Future Iterations

- `on_prompt_assembly` hook — per-message latency concern, needs optimization work
- Memory integration (`memory_tags` filtering, skill-guided writes) — separate techspec
- Cryptographic signing for marketplace — requires PKI infrastructure
- Skill auto-proposal (pattern detection + skillify meta-skill) — requires session analytics
- Override audit trail — useful but not blocking

---

## Architecture Decision Records

- [ADR-001: MCP Consent Model — Trust Tiers by Skill Source](adrs/adr-001.md) — Auto-approve bundled/user/additional/workspace MCP servers; marketplace requires explicit consent
- [ADR-002: Hybrid Hook Execution Model](adrs/adr-002.md) — Built-in hooks as Notifier callbacks, custom hooks as subprocess commands with fail-open semantics
- [ADR-003: Pluggable Registry Interface for Marketplace](adrs/adr-003.md) — Define marketplace.Registry interface, ship ClawHub as default implementation
- [ADR-004: Hash-Based Provenance for Marketplace Skills](adrs/adr-004.md) — SHA-256 hash verification + VerifyContent scanning, no cryptographic signing yet
