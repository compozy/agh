# TechSpec: Skills System — Increment 1 (Core)

## Executive Summary

AGH implements a skills system following the [AgentSkills](https://agentskills.io) open specification. Skills are prompt-first Markdown documents (`SKILL.md`) discovered from a four-level hierarchy (bundled → user → `.agents/` → workspace), injected as an XML catalog into each agent's system prompt, and loaded on demand via `agh skill view` CLI commands. No driver modification is required — the daemon controls the entire lifecycle through prompt assembly and CLI access.

This TechSpec covers **Increment 1 only**: SKILL.md loader & registry (F1), prompt injection (F2), security scanning (F4), CLI commands (F5), bundled skills (F9), and hot-reload (F10). Memory integration (F3), MCP lazy-load (F6), lifecycle hooks (F7), skill auto-proposal (F11), and ClawHub marketplace (F8) are deferred to Increment 2/3 TechSpecs with dedicated brainstorming.

**Primary trade-off**: prompt-only runtime sacrifices autonomous computation in exchange for zero execution infrastructure, inherent security, and cross-tool compatibility with 30+ AgentSkills-compliant agents.

**Key architectural decisions**: dual-scope registry with global + workspace layers (ADR-002), composed PromptAssembler with PromptProvider interface (ADR-003), stat-based polling for hot-reload (ADR-004), go:embed for bundled skills (ADR-005), `internal/skills/` as new package.

## System Architecture

### Component Overview

```
┌──────────────────────────────────────────────────────────────┐
│                       AGH Daemon                             │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                  internal/skills/                      │  │
│  │                                                        │  │
│  │  ┌───────────┐  ┌──────────┐  ┌──────────────────┐   │  │
│  │  │  Loader   │  │ Verify   │  │    Registry       │   │  │
│  │  │(SKILL.md  │  │(security │  │  global skills    │   │  │
│  │  │ parser)   │  │  scan)   │  │  (bundled + user) │   │  │
│  │  └─────┬─────┘  └────┬─────┘  │                   │   │  │
│  │        │              │        │  ForWorkspace(ws)  │   │  │
│  │        │              │        │  → merged snapshot │   │  │
│  │        │              │        └────────┬──────────┘   │  │
│  │        │              │                 │              │  │
│  │  ┌─────▼──────────────▼─────────────────▼──────────┐  │  │
│  │  │                  Watcher                         │  │  │
│  │  │  polls global dirs (3s); workspace dirs checked  │  │  │
│  │  │  lazily at prompt assembly via mtime comparison  │  │  │
│  │  └─────────────────────────────────────────────────┘  │  │
│  │                                                        │  │
│  │  ┌──────────────┐  ┌───────────────────────────────┐  │  │
│  │  │   Catalog    │  │  bundled/ (go:embed SKILL.md) │  │  │
│  │  │ (XML output) │  └───────────────────────────────┘  │  │
│  │  └──────┬───────┘                                     │  │
│  └─────────┼─────────────────────────────────────────────┘  │
│            │                                                 │
│  ┌─────────▼─────────────────────────────────────────────┐  │
│  │              daemon/ ComposedAssembler                 │  │
│  │   PromptProvider[] → memory | skills catalog           │  │
│  │   implements session.PromptAssembler                   │  │
│  └────────────────────────┬──────────────────────────────┘  │
│                           │ system prompt                    │
│  ┌────────────────────────▼──────────────────────────────┐  │
│  │              session.Manager                           │  │
│  └────────────────────────┬──────────────────────────────┘  │
│                           │                                  │
│  ┌──────────────────┐  ┌──▼──────────────────────────────┐  │
│  │  CLI             │  │  Agent Drivers (ACP)             │  │
│  │  agh skill ...   │  │  Claude, Codex, Gemini, Pi, ...  │  │
│  └──────────────────┘  └─────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

**Data flow:**

1. At boot, the daemon loads **global skills** (bundled + `~/.agh/skills/` + `~/.agents/skills/`) into the `Registry`
2. At prompt assembly, `CatalogProvider` receives the workspace path and calls `registry.ForWorkspace(workspace)` to merge global skills with **workspace skills** (`<workspace>/.agents/skills/` + `<workspace>/.agh/skills/`), returning a workspace-scoped snapshot
3. `ComposedAssembler` chains `memory.Assembler` and `skills.CatalogProvider` behind the existing `session.PromptAssembler` interface
4. At runtime, agents invoke `agh skill view <name>` via Bash to load full skill content
5. The `Watcher` polls global skill directories every 3s (configurable); workspace directories are checked lazily at prompt assembly time via mtime comparison
6. CLI commands (`agh skill list/view/info/create`) operate locally on the filesystem with no daemon connection — the daemon picks up changes via polling

## Implementation Design

### Core Interfaces

```go
// session/prompt_provider.go — new interface for composable prompt layers
type PromptProvider interface {
    PromptSection(ctx context.Context, workspace string) (string, error)
}
```

```go
// internal/skills/types.go — core domain types

// SkillMeta maps YAML frontmatter fields per AgentSkills spec.
type SkillMeta struct {
    Name        string            `yaml:"name"`
    Description string            `yaml:"description"`
    Version     string            `yaml:"version,omitempty"`
    Metadata    map[string]any    `yaml:"metadata,omitempty"`
}

// Skill is the complete in-memory representation.
type Skill struct {
    Meta     SkillMeta
    Content  string       // Markdown body after frontmatter
    Source   SkillSource
    Dir      string       // Absolute path to skill directory
    FilePath string       // Absolute path to SKILL.md
    Enabled  bool
}

// SkillSource tracks provenance for override precedence.
type SkillSource int
const (
    SourceBundled   SkillSource = iota // go:embed (lowest)
    SourceUser                         // ~/.agh/skills/ + ~/.agents/skills/
    SourceAgents                       // <workspace>/.agents/skills/
    SourceWorkspace                    // <workspace>/.agh/skills/ (highest)
)
```

```go
// internal/skills/registry.go — dual-scope skill registry

// Registry manages global skills (loaded at boot) and lazily merges
// workspace-scoped skills per session. Follows the memory.Store.ForWorkspace() pattern.
type Registry struct {
    mu              sync.RWMutex
    globalSkills    map[string]*Skill       // bundled + user-level
    workspaceCache  map[string]*wsCache     // keyed by workspace path
    globalVersion   atomic.Int64
    cfg             RegistryConfig
    logger          *slog.Logger
}

type wsCache struct {
    skills     map[string]*Skill
    snapshots  map[string]fileSnapshot  // mtime+size for staleness check
    lastAccess time.Time                // for TTL eviction (10 min)
}

func NewRegistry(cfg RegistryConfig, opts ...Option) *Registry
func (r *Registry) LoadAll(ctx context.Context) error   // load global skills at boot
func (r *Registry) Get(name string) (*Skill, bool)      // global-only lookup
func (r *Registry) List() []*Skill                       // global-only list
func (r *Registry) ForWorkspace(ctx context.Context, workspace string) ([]*Skill, error)
func (r *Registry) RefreshGlobal(ctx context.Context) error
func (r *Registry) GlobalVersion() int64
```

```go
// internal/skills/catalog.go — XML catalog for system prompt

// CatalogProvider implements session.PromptProvider.
// It calls registry.ForWorkspace() to produce a workspace-scoped catalog.
// No provider-level catalog caching in Inc 1 — ForWorkspace() handles
// its own mtime-based caching internally. Catalog string is rebuilt
// on every call (cheap: string formatting of skill names + descriptions).
type CatalogProvider struct {
    registry *Registry
}

func NewCatalogProvider(registry *Registry) *CatalogProvider
func (cp *CatalogProvider) PromptSection(ctx context.Context, workspace string) (string, error)
```

### Data Models

**RegistryConfig** — scanning configuration:

```go
type RegistryConfig struct {
    BundledFS       fs.FS    // fs.FS (not embed.FS) for testability
    UserSkillsDir   string   // ~/.agh/skills/
    UserAgentsDir   string   // ~/.agents/skills/
    DisabledSkills  []string
}
```

Note: `BundledFS` uses `fs.FS` interface (not concrete `embed.FS`) so that tests can supply `testing/fstest.MapFS`. The production `embed.FS` satisfies `fs.FS`.

Workspace directories (`<workspace>/.agents/skills/` and `<workspace>/.agh/skills/`) are not in the config — they are derived from the workspace path passed to `ForWorkspace()`.

**Warning** — security scan result:

```go
type Warning struct {
    Severity WarningSeverity
    Message  string
    Pattern  string
}

type WarningSeverity int
const (
    SeverityInfo     WarningSeverity = iota
    SeverityWarning
    SeverityCritical // Blocks skill loading
)
```

**fileSnapshot** — for stat-based polling and workspace cache staleness:

```go
type fileSnapshot struct {
    path    string
    modTime time.Time
    size    int64
}
```

### Dual-Scope Registry Design

The daemon is global but workspaces are per-session. The registry follows the same pattern as `memory.Store.ForWorkspace()`:

**Global layer** (loaded at boot, polled by Watcher):
- Bundled skills (go:embed) — immutable for process lifetime
- User skills: `~/.agh/skills/`, `~/.agents/skills/`

**Workspace layer** (loaded lazily, cached with staleness check):
- Project skills: `<workspace>/.agents/skills/`, `<workspace>/.agh/skills/`

`ForWorkspace(ctx, workspace)` does:
1. Check if cached workspace snapshot exists and is still valid (compare file mtime+size against cached snapshots)
2. If stale or missing: scan workspace skill directories, parse SKILL.md files, run VerifyContent, cache results
3. Merge: start with global skills map, overlay workspace skills (higher precedence wins on name collision)
4. Return merged `[]*Skill` sorted by name

This means:
- Sessions in different workspaces get different skill sets
- Workspace skills are discovered without daemon restart
- Cache invalidation is automatic via mtime check (no watcher needed for workspace dirs)
- The Watcher only polls global directories — workspace dirs are checked on-demand

### Prompt Assembly Pipeline

The current `memory.Assembler` owns the full prompt pipeline. It **prepends** memory context before the base prompt (see `assembler.go:80`: `contextBlock + "\n\n" + basePrompt`). The refactored pipeline must preserve this exact ordering.

**Before** (current — `memory/assembler.go:42`):
```
memory.Assembler.Assemble() → memory context + "\n\n" + agent.Prompt
```

**After** (refactored — preserves current ordering, appends skills):
```
daemon.ComposedAssembler.Assemble()
  → memory.Assembler.PromptSection() → memory context   (prepended, as today)
  → agent.Prompt (base system prompt from AgentDef)       (middle, as today)
  → skills.CatalogProvider.PromptSection() → skill catalog (appended, new)
  → concatenated result
```

**Final ordering**: `memory context → agent prompt → skill catalog`. This preserves current behavior (memory before prompt) and appends skills at the end where agents expect tool/skill instructions.

The `ComposedAssembler` lives in `daemon/` because it is composition logic — it wires providers but owns no domain logic. Each `PromptProvider` is independently testable.

**memory.Assembler refactoring**: The existing `Assemble()` method handles both memory loading and base prompt concatenation. The refactored version:
1. Keeps `Assemble()` for backward compatibility during transition
2. Adds `PromptSection()` method that returns only the memory context block (no base prompt)
3. `ComposedAssembler` handles the base prompt itself and calls `PromptSection()` on each provider — prepend providers run before the base prompt, append providers run after
4. **Regression test**: `ComposedAssembler` with only a memory provider must produce output byte-identical to current `memory.Assembler.Assemble()`

**Daemon boot independence**: The `ComposedAssembler` is constructed **regardless** of whether `cfg.Memory.Enabled` is true or false. When memory is disabled, the memory provider is nil/omitted from the chain. When skills are disabled, the skills provider is nil/omitted. The assembler handles zero, one, or many providers gracefully. This prevents the current bug where `promptAssembler` is only set inside the `cfg.Memory.Enabled` branch.

### Catalog XML Format

```xml
<available-skills>
  <skill name="agh-session-guide">Manage AGH sessions: create, monitor, stop, resume via CLI</skill>
  <skill name="agh-memory-guide">Use AGH persistent memory: scopes, CLI commands, consolidation</skill>
  <skill name="agh-agent-setup">Configure AGH agents: AGENT.md format, providers, MCP servers</skill>
</available-skills>

Use `agh skill view <name>` to load full instructions for any skill.
Use `agh skill view <name> --file <path>` to read a specific skill resource file.
```

Descriptions are truncated at 200 characters. Skills are sorted alphabetically by name.

### `agh skill view` Output Format

Output uses **XML-like delimiters for LLM consumption**, not strict XML. The Markdown body is included verbatim (not escaped) between tags. This matches the convention used by Claude Code, OpenClaw, and other AgentSkills-compliant tools — agents parse these via string matching, not XML parsers.

```
<skill_content name="agh-session-guide">
# Session Guide

Manage AGH sessions using the CLI:
1. Create a session: `agh session new --agent <name>`
2. List sessions: `agh session list`
...

<skill_resources>
  <file>references/session-types.md</file>
</skill_resources>
</skill_content>
```

The output is plain text piped to stdout. No Content-Type header, no encoding declaration. Agents consume it via their Bash/shell tool output.

### API Endpoints

No new HTTP API endpoints. Skills are managed exclusively via CLI commands. The catalog is injected into the system prompt via the composed assembler. The existing dashboard API is unaffected.

**CLI Commands** (registered under `agh skill`):

| Command | Args | Description |
|---------|------|-------------|
| `agh skill list` | `[--source <source>]` | List installed skills with name, description, source, enabled status |
| `agh skill view <name>` | `[--file <path>]` | Return skill body in XML or a specific resource file |
| `agh skill info <name>` | — | Show full metadata, source, path, and resource listing |
| `agh skill create [name]` | — | Scaffold new skill directory with SKILL.md template |

Marketplace commands (`install`, `remove`, `search`) are deferred to Increment 3.

**CLI commands are local-only**: All skill CLI commands operate directly on the filesystem. They do NOT connect to the daemon. `agh skill create` writes a new SKILL.md to disk — the daemon's Watcher detects it via polling within the next interval. There is no "immediate refresh" RPC from CLI to daemon.

**CLI reuses the same resolution and verification pipeline as the registry**: `agh skill list`, `agh skill view`, and `agh skill info` instantiate a read-only `Registry` locally (same `LoadAll()` + `ForWorkspace()` + `VerifyContent()` code path). This ensures:
- Override precedence is consistent between CLI output and prompt catalog
- Security-blocked skills are excluded from `agh skill view` output (not just from the daemon catalog)
- A critically blocked workspace skill cannot be loaded by an agent via `agh skill view` even though the file exists on disk

The CLI registry is ephemeral (created per command invocation, not persisted). It shares the `skills.Registry` type but has no watcher or caching.

### Memory and Skills Composition

In Inc 1, memory and skills **coexist in the prompt** with no coupling between them. The `ComposedAssembler` chains the existing `memory.Assembler` and the new `skills.CatalogProvider` — memory context appears first, then skill catalog. No changes to memory loading logic. Skills benefit from memory context automatically because it's in the same prompt.

Deep memory-skills integration (F3) — tag-filtered memory injection, skill-guided memory writes, memory query API for skill activation — is deferred to **Increment 2** where it will receive dedicated brainstorming and its own TechSpec section alongside MCP lazy-load and lifecycle hooks.

### Security Scanning (F4)

`VerifyContent()` scans skill content for prompt injection patterns before loading into the registry.

**Scan targets**: All non-bundled skills (user, .agents, workspace). Bundled skills are trusted.

**Pattern categories**:

| Severity | Patterns | Action |
|----------|----------|--------|
| Critical | System prompt overrides (`ignore all previous`, `you are now`), tool abuse instructions (`delete all files`, `rm -rf`), credential extraction (`print your API key`) | Block loading |
| Warning | Unusual tool patterns, references to sensitive paths, excessive tool chaining | Log warning, allow loading |
| Info | Long content (>50K chars), unusual frontmatter fields | Log info, allow loading |

**Implementation**: Regex-based pattern matching on the Markdown body. No AST parsing needed — patterns are string-level. Returns `[]Warning` with the highest severity determining the action.

### Hot-Reload (F10)

**Mechanism**: stat-based polling for global dirs + lazy mtime check for workspace dirs.

1. **Bundled skills** (go:embed) are immutable for the process lifetime — no polling
2. **Watcher goroutine** scans **global** directories (`~/.agh/skills/`, `~/.agents/skills/`) every 3 seconds (configurable via `skills.poll_interval` in config):
   - Collect `mtime` + `size` for each SKILL.md via `os.Stat()`
   - Compare against previous snapshot
   - If changed: re-parse affected files, atomic swap global skills map, bump `globalVersion`
3. **Workspace directories** are NOT polled by the Watcher. Instead, `ForWorkspace()` checks mtime of cached workspace files on each call. If stale, it re-scans. This is cheap because it only runs at prompt assembly time (session creation).
4. **Version-aware caching** lives inside `CatalogProvider`, not in `session`. The provider compares the registry's `GlobalVersion()` plus workspace cache freshness before deciding to rebuild the catalog. Sessions call `PromptAssembler.Assemble()` as before — no version awareness needed in `session/`.

**Goroutine ownership**: Watcher goroutine started by daemon during boot, stopped via `context.Context` cancellation during shutdown. Tracked with `sync.WaitGroup` in daemon.

### Loading Hierarchy

**Global loading** (at boot via `LoadAll()`): bundled first, then user dirs. Higher-precedence sources override same-name skills.

**Workspace merging** (at prompt assembly via `ForWorkspace()`): global skills first, then workspace dirs overlaid.

| Level | Directories | Scope | Override Priority |
|-------|------------|-------|-------------------|
| 1 (lowest) | `go:embed bundled/skills/` | Global | Default set |
| 2 | `~/.agh/skills/` + `~/.agents/skills/` | Global | User global |
| 3 | `<workspace>/.agents/skills/` | Per-workspace | Project cross-client |
| 4 (highest) | `<workspace>/.agh/skills/` | Per-workspace | Project AGH-specific |

Constraints:
- Max directory depth: 4 levels
- Max candidates per root: 300
- Skip `.git/`, `node_modules/`, hidden dirs (except `.agh/`, `.agents/`)
- Log warning on name collisions with override source info

### Bundled Skills (F9)

Ship 3 starter skills via `go:embed`:

| Skill | Purpose |
|-------|---------|
| `agh-session-guide` | How to create, manage, and monitor AGH sessions via CLI |
| `agh-memory-guide` | How to use AGH persistent memory (scopes, CLI, consolidation) |
| `agh-agent-setup` | How to configure agents (AGENT.md format, providers, MCP servers) |

Each is a standard SKILL.md file in `internal/skills/bundled/skills/<name>/SKILL.md`. Parsed using the same `ParseSkillFile()` path as filesystem skills. Users can override bundled skills by placing a same-name skill in any higher-precedence directory.

## Integration Points

### Daemon Boot Sequence

**Integration point**: `internal/daemon/daemon.go`, `boot()` function (line 540).

**Current sequence** (relevant excerpt):
```
... → memory store + assembler (inside cfg.Memory.Enabled branch) → lock → registry → session manager → ...
```

**Modified sequence**:
```
... → memory store (if enabled) → skills registry (if enabled) → composed assembler (always) → lock → registry → session manager → ...
```

**Critical fix**: The `ComposedAssembler` is constructed **unconditionally**, outside any feature-flag branch. It collects whichever providers are enabled:

```go
var providers []session.PromptProvider

// Memory provider (optional)
if cfg.Memory.Enabled {
    memoryStore = memory.NewStore(globalMemoryDir)
    memAssembler := memory.NewAssembler(memoryStore)
    providers = append(providers, memAssembler)
}

// Skills provider (optional)
if cfg.Skills.Enabled {
    skillsRegistry := skills.NewRegistry(skillsCfg, skills.WithLogger(logger))
    skillsRegistry.LoadAll(ctx)
    watcher := skills.NewWatcher(skillsRegistry, pollInterval)
    go watcher.Start(ctx)
    providers = append(providers, skills.NewCatalogProvider(skillsRegistry))
}

// Always construct the composed assembler
promptAssembler := NewComposedAssembler(providers...)
```

This ensures `memory.enabled=false` + `skills.enabled=true` works correctly, and vice versa.

### Config (internal/config)

**New section** in `Config` struct (follows `DreamConfig` pattern for `time.Duration`):

```go
// SkillsConfig controls skill loading and discovery.
type SkillsConfig struct {
    Enabled        bool          `toml:"enabled"`
    DisabledSkills []string      `toml:"disabled_skills,omitempty"`
    PollInterval   time.Duration `toml:"poll_interval"` // default 3s
}
```

Default: `enabled = true`, `poll_interval = 3 * time.Second`.

**Merge overlay** (added to `merge.go` alongside existing overlays):

```go
type skillsOverlay struct {
    Enabled        *bool          `toml:"enabled"`
    DisabledSkills *[]string      `toml:"disabled_skills"`
    PollInterval   *time.Duration `toml:"poll_interval"`
}
```

Added to `configOverlay` struct and wired in `configOverlay.Apply()`. This is required because `merge.go` uses strict TOML decoding that rejects unknown keys — the `[skills]` section must be declared in the overlay structs.

**New field** in `HomePaths`:
```go
SkillsDir string // ~/.agh/skills/
```

### CLI (internal/cli)

**New subcommand**: `newSkillCommand(deps)` registered in `root.go` alongside existing commands.

**Dependencies**: `commandDeps` already provides `loadConfig`, `resolveHome` — skill commands use these to resolve directories and scan locally.

**All skill commands are local-only**: They scan skill directories on the filesystem directly, parse SKILL.md files, and output results. No daemon client (`newClient`) needed. This is a key difference from memory commands which go through the daemon. The daemon picks up filesystem changes via its Watcher (global dirs) or lazy mtime check (workspace dirs).

`agh skill view` outputs to stdout in XML format for agent consumption. Strips YAML frontmatter, wraps content in `<skill_content>` tags, lists resource files.

### Existing Package Changes

| Package | Change |
|---------|--------|
| `session/` | Add `PromptProvider` interface (new file: `prompt_provider.go`) |
| `memory/` | Add `PromptSection()` method to `Assembler` (returns memory block only, no base prompt) |
| `daemon/` | Add `ComposedAssembler` struct, wire skills in `boot()` unconditionally, manage watcher lifecycle |
| `config/` | Add `SkillsConfig` to `Config`, `skillsOverlay` to `merge.go`, `SkillsDir` to `HomePaths`, update `EnsureHomeLayout` |
| `cli/` | Add `newSkillCommand()` with list/view/info/create subcommands (local-only, no daemon client) |

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/skills/` | New | New package with 7 files. No risk — isolated module. | Implement from scratch |
| `internal/skills/bundled/` | New | Embedded skills via go:embed. No risk — static assets. | Create SKILL.md files + embed.go |
| `session/prompt_provider.go` | New | New `PromptProvider` interface. No risk — additive. | Define interface |
| `memory/assembler.go` | Modified | Add `PromptSection()` method. Low risk — existing `Assemble()` unchanged. | Add method, extract memory-only logic |
| `daemon/daemon.go` | Modified | Refactor boot to construct ComposedAssembler unconditionally. Medium risk — changes control flow in core boot path. | Extract assembler construction from Memory.Enabled branch, add skills wiring |
| `daemon/composed_assembler.go` | New | Composed prompt assembler. Low risk — simple composition. | Implement PromptAssembler via PromptProvider chain |
| `config/config.go` | Modified | Add `SkillsConfig` section. Low risk — additive TOML field. | Add struct + default values |
| `config/merge.go` | Modified | Add `skillsOverlay` struct and wiring. Low risk — follows existing overlay pattern. | Add overlay struct, wire in `configOverlay.Apply()` |
| `config/home.go` | Modified | Add `SkillsDir` to `HomePaths`. Low risk — additive. | Add field + ensure directory in layout |
| `cli/root.go` | Modified | Register `agh skill` subcommand. Low risk — additive. | Add `cmd.AddCommand(newSkillCommand(deps))` |
| `cli/skill.go` | New | Skill CLI commands (local-only). No risk — new file. | Implement list/view/info/create |
| `go.mod` | Modified | Add `gopkg.in/yaml.v3`. Low risk — well-maintained dependency. | `go get gopkg.in/yaml.v3` |
| Existing drivers | No change | System prompt already passed to drivers. | None |
| Web dashboard | No change | Dashboard is unaffected. | None |

## Testing Approach

### Unit Tests

- **Loader** (`loader_test.go`): Parse valid SKILL.md with all frontmatter fields; handle malformed YAML; handle missing frontmatter; handle empty body; validate name constraints; test lenient parsing fallback; test `metadata.agh` extension parsing
- **Registry** (`registry_test.go`): LoadAll from global directories; verify override precedence (user > bundled); ForWorkspace merges global + workspace skills; workspace skill overrides global same-name skill; workspace cache invalidation on mtime change; concurrent read access under `RWMutex`; globalVersion increments correctly
- **Catalog** (`catalog_test.go`): XML generation format; description truncation at 200 chars; escaping of special characters in names/descriptions; empty skill list produces empty string; alphabetical sorting
- **CatalogProvider** (`catalog_provider_test.go`): Returns empty string when no skills; returns valid XML for skill set; caches catalog and rebuilds only on version change; produces different catalogs for different workspaces
- **Verify** (`verify_test.go`): Detect critical injection patterns (blocks loading); detect warning patterns (allows loading); pass clean content; severity ordering; edge cases (empty content, very long content)
- **Watcher** (`watcher_test.go`): Detect new SKILL.md in global dirs; detect modified SKILL.md (mtime change); detect deleted SKILL.md; no false positive when mtime unchanged; context cancellation stops polling
- **ComposedAssembler** (`composed_assembler_test.go`): Chains providers in order; handles zero providers (returns base prompt only); handles nil providers; handles provider errors; passes correct workspace to each provider

Mock requirements:
- `fs.FS` for bundled skills (use `testing/fstest.MapFS`)
- Temporary directories for filesystem operations (`t.TempDir()`)
- Mock `PromptProvider` for composed assembler tests

### Integration Tests

- **Boot integration**: Daemon starts with skills loaded; verify registry populated with bundled skills; verify composed assembler produces prompt with memory + skills sections; verify boot works with `memory.enabled=false` + `skills.enabled=true`
- **CLI integration**: `agh skill list` returns expected skills; `agh skill view <name>` returns formatted XML; `agh skill create` scaffolds valid SKILL.md; `agh skill info` shows metadata
- **Override integration**: Workspace skill overrides bundled skill with same name; verify override logged
- **Multi-workspace integration**: Two sessions with different workspaces get different skill catalogs
- **Hot-reload integration**: Add SKILL.md to watched global directory; verify registry picks it up within poll interval; delete skill; verify removal
- **Prompt assembly integration**: Create session with skills enabled; verify agent receives system prompt containing skill catalog

Test data: Fixture SKILL.md files with various frontmatter combinations in `testdata/` directories.

## Development Sequencing

### Build Order

1. **`internal/skills/types.go`** — Define `Skill`, `SkillMeta`, `SkillSource`, `Warning`, `WarningSeverity`, `RegistryConfig`, `fileSnapshot`. No dependencies.
2. **`internal/skills/loader.go`** — `ParseSkillFile()`, `parseFrontmatter()`, `scanDirectory()`. Depends on step 1. Requires `go get gopkg.in/yaml.v3`.
3. **`internal/skills/verify.go`** — `VerifyContent()` with regex pattern matching. Depends on step 1.
4. **`internal/skills/registry.go`** — `Registry` with `LoadAll()`, `Get()`, `List()`, `ForWorkspace()`, `RefreshGlobal()`, `GlobalVersion()`. Depends on steps 2, 3.
5. **`internal/skills/catalog.go`** — `BuildCatalog()` XML string builder + `CatalogProvider` with version-aware caching. Depends on steps 1, 4.
6. **`internal/skills/bundled/`** — `embed.go` with `//go:embed` directive + 3 starter SKILL.md files. Depends on step 1.
7. **`internal/skills/watcher.go`** — `Watcher` with stat-based polling loop for global dirs only. Depends on step 4.
8. **`session/prompt_provider.go`** — `PromptProvider` interface. No dependencies on skills package.
9. **`memory/assembler.go` refactor** — Add `PromptSection()` method. Depends on step 8.
10. **`config/` changes** — Add `SkillsConfig`, `SkillsDir` to `HomePaths`. No dependencies on skills package.
11. **`daemon/composed_assembler.go`** — `ComposedAssembler` implementing `PromptAssembler`. Depends on steps 8, 9.
12. **`daemon/daemon.go` integration** — Refactor boot to construct ComposedAssembler unconditionally, wire registry + watcher + skills provider. Depends on steps 4, 5, 7, 10, 11.
13. **`cli/skill.go`** — `agh skill list/view/info/create` commands (local-only). Depends on steps 2, 4, 5.

### Technical Dependencies

- **`gopkg.in/yaml.v3`**: Required for YAML frontmatter parsing. Install via `go get`.
- **No other new dependencies**: Polling uses stdlib `os.Stat`. XML output is string formatting. Embed uses stdlib `embed`. Concurrency uses stdlib `sync`.

## Monitoring and Observability

All logging uses `log/slog` per project conventions:

| Event | Level | Structured Fields |
|-------|-------|-------------------|
| Skill loaded | Info | `name`, `source`, `dir` |
| Skill skipped (parse error) | Warn | `path`, `error` |
| Skill blocked (security) | Warn | `name`, `pattern`, `severity` |
| Skill override | Info | `name`, `overridden_source`, `new_source` |
| Skill disabled | Debug | `name` |
| Global registry refreshed | Info | `skill_count`, `version`, `changed` |
| Workspace skills loaded | Debug | `workspace`, `skill_count`, `cached` |
| Watcher started | Info | `roots`, `interval` |
| Watcher detected change | Debug | `path`, `action` (added/modified/deleted) |
| Catalog built | Debug | `skill_count`, `catalog_bytes`, `workspace` |
| Skill view requested | Debug | `name`, `file` |

## Technical Considerations

### Key Decisions

See Architecture Decision Records below for full details.

- **Three-increment delivery** (ADR-001): Core → MCP+Hooks → Marketplace as independent increments.
- **Dual-scope registry** (ADR-002): Global skills at boot + workspace skills lazily merged per-session, following the `memory.Store.ForWorkspace()` pattern.
- **Composed PromptAssembler** (ADR-003): Refactor prompt assembly into PromptProvider chain. Memory and skills are independent providers wired in daemon/. Constructed unconditionally to avoid feature-flag coupling.
- **Stat-based polling** (ADR-004): Default 3s polling (configurable via `skills.poll_interval`) for global dirs. Workspace dirs checked lazily via mtime. No fsnotify — correctness over speed.
- **go:embed bundled skills** (ADR-005): Bundled skills are real SKILL.md files parsed by the same loader. Uses `fs.FS` interface for testability.

### Known Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Large skill count inflates system prompt | Medium | Cap catalog at 150 skills, truncate descriptions at 200 chars. Warn when approaching limits. |
| Workspace skills contain prompt injection | Medium | Mandatory `VerifyContent()` scan. Critical patterns block loading. |
| YAML parsing edge cases from other tools | Medium | Lenient parser: warn on issues, skip only if name missing or YAML unparseable. |
| Agent doesn't have Bash access for `agh skill view` | Very Low | All current drivers provide Bash/shell tools. Document requirement. |
| Polling overhead with many skill directories | Low | Cap at 300 candidates per root. Skip `.git/`, `node_modules/`. Stat calls are cheap on local SSD. |
| PromptAssembler refactoring breaks existing behavior | Low | Ordering preserved: `memory context → agent prompt → skills catalog`. Regression test: ComposedAssembler with memory-only provider must produce byte-identical output to current `memory.Assembler.Assemble()`. |
| Workspace cache grows unbounded with many workspaces | Low | Evict workspace cache entries older than 10 minutes. Daemon serves few concurrent workspaces in practice. |

### Scope Boundaries

**In scope (Increment 1)**:
- SKILL.md loader with YAML frontmatter parsing
- Dual-scope registry: global skills at boot + workspace skills lazily merged per-session
- XML catalog injection via composed PromptAssembler (constructed unconditionally)
- Security scanning (VerifyContent) for non-bundled skills
- CLI: list, view, info, create (local-only, no daemon connection)
- 3 bundled skills via go:embed
- Stat-based hot-reload: polling for global dirs, mtime check for workspace dirs
- Config section for skills (enabled, disabled_skills, poll_interval)

**Out of scope (deferred to Inc 2/3 TechSpecs)**:
- Memory integration: tag-filtered injection, skill-guided writes, memory query API (Inc 2)
- MCP lazy-load from skill frontmatter (Inc 2)
- Lifecycle hooks: on_session_created, on_session_stopped, on_prompt_assembly (Inc 2)
- Skill auto-proposal and skillify meta-skill (Inc 2)
- ClawHub marketplace: install, remove, search (Inc 3)
- Cryptographic provenance verification (Inc 3)
- Auto-activation via `when_to_use` or `paths` matching
- Skill versioning and pinning
- Private registry / enterprise RBAC

## Architecture Decision Records

- [ADR-001: Three-Increment Delivery Strategy](adrs/adr-001.md) — Ship core → MCP+hooks → marketplace as independent increments
- [ADR-002: Dual-Scope Registry](adrs/adr-002.md) — Global skills at boot + workspace skills lazily merged per-session
- [ADR-003: Composed PromptAssembler with PromptProvider Interface](adrs/adr-003.md) — Refactor prompt assembly into composable PromptProvider chain wired in daemon/
- [ADR-004: Stat-Based Polling for Hot-Reload](adrs/adr-004.md) — Polling + version counter over fsnotify for cross-platform reliability
- [ADR-005: go:embed for Bundled Skills](adrs/adr-005.md) — Real SKILL.md files embedded into binary, parsed by same loader via fs.FS
