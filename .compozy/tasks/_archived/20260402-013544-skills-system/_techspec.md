# TechSpec: Skills System

## Executive Summary

AGH implements a skills system following the [AgentSkills](https://agentskills.io) open specification. Skills are prompt-only Markdown documents (`SKILL.md`) discovered from a four-level hierarchy (bundled → user → `.agents/` → workspace), injected as an XML catalog into each agent's system prompt, and loaded on demand via `agh skill view` CLI commands. No driver modification is required — the kernel controls the entire lifecycle through prompt assembly and CLI access.

Key architectural decisions: prompt-only runtime (ADR-001), SKILL.md native format (ADR-002), system prompt injection with CLI access pattern (ADR-003), and four-level loading hierarchy (ADR-004).

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────┐
│                    AGH Kernel                           │
│                                                         │
│  ┌───────────────────────────────────────────────────┐  │
│  │              internal/skills/                     │  │
│  │                                                   │  │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │  │
│  │  │ Registry │  │  Loader  │  │  Eligibility  │  │  │
│  │  │(in-memory│  │(SKILL.md │  │  (OS, bins,   │  │  │
│  │  │ HashMap) │  │ parser)  │  │   env, roles) │  │  │
│  │  └────┬─────┘  └────┬─────┘  └───────┬───────┘  │  │
│  │       │              │                │           │  │
│  │  ┌────▼──────────────▼────────────────▼───────┐  │  │
│  │  │              Snapshot Builder               │  │  │
│  │  │     (XML catalog for system prompt)        │  │  │
│  │  └────────────────────┬───────────────────────┘  │  │
│  │                       │                           │  │
│  │  ┌────────────────────▼───────────────────────┐  │  │
│  │  │              Catalog Builder                │  │  │
│  │  │  (formats <available_skills> XML block)    │  │  │
│  │  └────────────────────┬───────────────────────┘  │  │
│  │                       │                           │  │
│  │  ┌──────────┐  ┌──────▼──────┐  ┌────────────┐  │  │
│  │  │  Verify  │  │  ClawHub    │  │  Bundled    │  │  │
│  │  │(security)│  │  (marketplace│  │  (go:embed) │  │  │
│  │  └──────────┘  │   client)   │  └────────────┘  │  │
│  │                 └─────────────┘                   │  │
│  └───────────────────────────────────────────────────┘  │
│                                                         │
│  ┌──────────────────┐  ┌─────────────────────────────┐  │
│  │  Prompt Assembly  │  │  CLI (agh skill ...)        │  │
│  │  (4-layer system) │  │  list/view/info/install/    │  │
│  │                    │  │  remove/search/create       │  │
│  └──────────────────┘  └─────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
         │                              │
         │ system prompt                │ agh skill view
         ▼                              ▼
┌─────────────────────────────────────────────────────────┐
│                    Agent Drivers                        │
│  ┌────────┐  ┌────────┐  ┌──────────┐  ┌───────────┐  │
│  │ Claude │  │ Codex  │  │ OpenCode │  │    Pi     │  │
│  └────────┘  └────────┘  └──────────┘  └───────────┘  │
└─────────────────────────────────────────────────────────┘
```

**Data flow:**

1. At boot, the kernel scans skill directories and populates the `Registry`
2. At agent spawn, a `SkillSnapshot` is built from eligible skills and the XML catalog is injected into the system prompt
3. At runtime, agents invoke `agh skill view <name>` via Bash to load full skill content
4. Optionally, agents invoke `agh skill view <name> --file <path>` to read specific resources

## Implementation Design

### Core Interfaces

```go
// internal/skills/types.go

// SkillMeta maps YAML frontmatter fields per AgentSkills spec.
type SkillMeta struct {
    Name          string            `yaml:"name"`
    Description   string            `yaml:"description"`
    License       string            `yaml:"license"`
    Compatibility string            `yaml:"compatibility"`
    AllowedTools  string            `yaml:"allowed-tools"`
    Metadata      map[string]string `yaml:"metadata"`
}

// Skill is the complete in-memory representation.
type Skill struct {
    Meta     SkillMeta
    Content  string      // Markdown body (after frontmatter)
    Source   SkillSource
    Dir      string      // Absolute path to skill directory
    FilePath string      // Absolute path to SKILL.md
    Enabled  bool
}

// SkillSource tracks provenance for override precedence.
type SkillSource int
const (
    SourceBundled   SkillSource = iota
    SourceUser
    SourceAgents
    SourceWorkspace
    SourceClawHub
)
```

```go
// internal/skills/registry.go

// Registry manages all loaded skills with thread-safe access.
type Registry struct {
    mu       sync.RWMutex
    skills   map[string]*Skill
    snapshot *SkillSnapshot
    version  int
    frozen   bool
}

func NewRegistry() *Registry
func (r *Registry) LoadAll(cfg LoadConfig) error
func (r *Registry) Get(name string) (*Skill, bool)
func (r *Registry) List() []*Skill
func (r *Registry) Snapshot(filter SnapshotFilter) *SkillSnapshot
func (r *Registry) Freeze()
```

```go
// internal/skills/loader.go

// ParseSkillFile reads a SKILL.md and returns a Skill.
func ParseSkillFile(path string) (*Skill, error)

// parseFrontmatter extracts YAML frontmatter from SKILL.md content.
func parseFrontmatter(content string) (SkillMeta, string, error)
```

```go
// internal/skills/catalog.go

// BuildCatalog generates the XML catalog string for system prompt injection.
func BuildCatalog(skills []*Skill) string
```

```go
// internal/skills/verify.go

// VerifyContent scans skill content for prompt injection patterns.
func VerifyContent(content string) []Warning
```

```go
// internal/skills/clawhub.go

// Client interacts with the ClawHub marketplace API.
type Client struct {
    baseURL    string
    httpClient *http.Client
}

func NewClient() *Client
func (c *Client) Search(ctx context.Context, query string) ([]SkillListing, error)
func (c *Client) Install(ctx context.Context, slug, targetDir string) error
```

### Data Models

**SkillSnapshot** — immutable view for prompt assembly:

```go
type SkillSnapshot struct {
    Skills  []*Skill // Filtered eligible skills
    Catalog string   // Pre-formatted XML catalog
    Version int      // Monotonically increasing
}
```

**LoadConfig** — scanning configuration:

```go
type LoadConfig struct {
    BundledFS      embed.FS
    UserDir        string   // ~/.agh/skills/
    UserAgentsDir  string   // ~/.agents/skills/
    AgentsDir      string   // <workspace>/.agents/skills/
    WorkspaceDir   string   // <workspace>/.agh/skills/
    DisabledSkills []string
}
```

**SnapshotFilter** — controls which skills enter the snapshot:

```go
type SnapshotFilter struct {
    AllowedSkills []string // Empty = all skills
    OS            string   // runtime.GOOS
}
```

**Warning** — security scan result:

```go
type Warning struct {
    Severity WarningSeverity // Info, Warning, Critical
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

**SkillListing** — ClawHub search result:

```go
type SkillListing struct {
    Slug        string
    Name        string
    Description string
    Author      string
    Downloads   int
    Stars       int
}
```

### API Endpoints

No new HTTP API endpoints. Skills are managed exclusively via CLI commands, and the catalog is injected into the system prompt. The existing dashboard API is unaffected.

**CLI Commands** (registered under `agh skill`):

| Command | Args | Description |
|---------|------|-------------|
| `agh skill list` | `[--source <source>]` | List installed skills with name, description, source, enabled status |
| `agh skill view <name>` | `[--file <path>]` | Return skill body in `<skill_content>` XML or a specific file |
| `agh skill info <name>` | — | Show full metadata, source, path, and resource listing |
| `agh skill install <slug>` | `[--from github]` | Install from ClawHub (default) or GitHub |
| `agh skill remove <name>` | — | Delete skill directory from filesystem |
| `agh skill search <query>` | `[--limit N]` | Search ClawHub marketplace |
| `agh skill create [name]` | — | Scaffold new skill directory with SKILL.md template |

**`agh skill view` output format:**

```xml
<skill_content name="code-review">
# Code Review

When reviewing code, follow these steps:
1. Check for security vulnerabilities
2. Verify error handling
...

<skill_resources>
  <file>scripts/lint-check.sh</file>
  <file>references/security-checklist.md</file>
</skill_resources>
</skill_content>
```

## Integration Points

### ClawHub Marketplace

- **Purpose**: Community skill discovery and installation
- **Base URL**: `https://clawhub.ai/api/v1/`
- **Endpoints used**: `GET /search?q=...`, `GET /download?slug=...`
- **Authentication**: None required for public skills
- **Error handling**: Exponential backoff (1.5s initial → 30s max, 5 retries)
- **Security**: All downloaded skills pass through `VerifyContent()` before installation

### Kernel Prompt Assembly

- **Integration point**: `internal/prompt/builder.go` (existing)
- **Change**: Add skills catalog as 4th layer in `AssemblePrompt()`
- **Prompt order**: Template → Role → **Skills Catalog** → Context

### Kernel Boot Sequence

- **Integration point**: `internal/kernel/kernel.go`
- **Change**: Add `load_skills` step after `load_prompt_templates` (step 10)
- **Behavior**: `skills.NewRegistry()` → `LoadAll()` → `Freeze()`

### Agent Spawn

- **Integration point**: `internal/kernel/session_manager.go`
- **Change**: Build `SkillSnapshot` before prompt assembly
- **Behavior**: Filter eligible skills → build catalog XML → include in system prompt

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/skills/` | New | New package with 7 files. No risk — isolated module. | Implement from scratch |
| `internal/prompt/builder.go` | Modified | Add skills catalog as 4th prompt layer. Low risk — additive change. | Add `SkillSnapshot` to `AssembleOpts`, insert catalog between role and context |
| `internal/kernel/kernel.go` | Modified | Add `skillRegistry` field and boot step. Low risk — additive. | Add field, initialize in boot sequence |
| `internal/kernel/session_manager.go` | Modified | Build skill snapshot during agent spawn. Low risk — before prompt assembly. | Call `registry.Snapshot()`, pass to prompt builder |
| `internal/cli/` | Modified | Add `skill` subcommand group with 7 commands. Low risk — new command tree. | Register cobra commands |
| `go.mod` | Modified | Add YAML parsing dependency. Low risk. | `go get gopkg.in/yaml.v3` |
| Existing drivers | No change | System prompt already passed to drivers. Zero driver modification. | None |
| Web dashboard | No change | Dashboard is unaffected. | None |

## Testing Approach

### Unit Tests

- **Loader** (`loader_test.go`): Parse valid SKILL.md, handle malformed YAML, handle missing frontmatter, handle empty body, validate name constraints, test lenient parsing fallback
- **Registry** (`registry_test.go`): Load from multiple directories, verify override precedence, test name collision logging, test snapshot generation, test freeze behavior, concurrent read access
- **Eligibility** (`eligibility_test.go`): OS filtering, disabled skills, allowlist filtering
- **Catalog** (`catalog_test.go`): XML generation format, escaping of special characters in names/descriptions, empty skills list produces empty string
- **Verify** (`verify_test.go`): Detect critical injection patterns, detect warning patterns, pass clean content
- **ClawHub** (`clawhub_test.go`): Search response parsing, download and extraction, retry logic, error handling

Mock requirements:
- `embed.FS` for bundled skills
- HTTP test server for ClawHub client
- Temporary directories for filesystem operations

### Integration Tests

- **Boot integration**: Kernel starts with skills loaded, verify snapshot populated
- **Spawn integration**: Agent receives system prompt containing skill catalog
- **CLI integration**: `agh skill list`, `agh skill view`, `agh skill install` end-to-end
- **Override integration**: Workspace skill overrides bundled skill with same name

Test data: fixture SKILL.md files with various frontmatter combinations in `testdata/` directories.

## Development Sequencing

### Build Order

1. **`internal/skills/types.go`** — Define all types (no dependencies)
2. **`internal/skills/loader.go`** — SKILL.md parser (depends on types, needs `gopkg.in/yaml.v3`)
3. **`internal/skills/verify.go`** — Security scanning (depends on types)
4. **`internal/skills/eligibility.go`** — Filtering logic (depends on types)
5. **`internal/skills/registry.go`** — Registry with loading pipeline (depends on loader, verify, eligibility)
6. **`internal/skills/catalog.go`** — XML catalog builder (depends on types)
7. **`internal/skills/bundled/`** — Embedded skills via `go:embed` (depends on types)
8. **Kernel integration** — Add registry to kernel boot + snapshot to agent spawn (depends on registry, catalog)
9. **Prompt assembly** — Add skills layer to `AssemblePrompt` (depends on catalog)
10. **CLI commands** — `agh skill list/view/info/create` (depends on registry)
11. **`internal/skills/clawhub.go`** — ClawHub client (depends on types, verify)
12. **CLI marketplace commands** — `agh skill install/remove/search` (depends on clawhub, registry)

### Technical Dependencies

- **`gopkg.in/yaml.v3`**: Required for YAML frontmatter parsing. Install via `go get`.
- **No other new dependencies**: HTTP client uses stdlib `net/http`. Embed uses stdlib `embed`. XML output is string formatting — no XML library needed.

## Monitoring and Observability

All logging uses `log/slog` per project conventions:

| Event | Level | Structured Fields |
|-------|-------|-------------------|
| Skill loaded | Info | `name`, `source`, `dir` |
| Skill skipped (parse error) | Warn | `path`, `error` |
| Skill blocked (security) | Warn | `name`, `pattern`, `severity` |
| Skill override | Info | `name`, `overridden_source`, `new_source` |
| Skill disabled | Debug | `name` |
| Registry frozen | Info | `skill_count`, `version` |
| ClawHub search | Debug | `query`, `result_count` |
| ClawHub install | Info | `slug`, `target_dir` |
| ClawHub error | Error | `slug`, `error`, `retry_count` |

## Technical Considerations

### Key Decisions

See Architecture Decision Records below for full details.

- **Prompt-only runtime** (ADR-001): Skills are Markdown instructions, not executable code. Agents use their existing tools to carry out instructions.
- **SKILL.md native format** (ADR-002): 100% AgentSkills spec compliance. Single parser. Cross-tool interoperability.
- **System prompt + CLI access** (ADR-003): Zero driver modification. Catalog in prompt, full content via `agh skill view`.
- **Four-level loading** (ADR-004): Bundled → user → .agents → workspace with override semantics.

### Known Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Large number of skills inflates system prompt | Medium | Cap catalog at 150 skills, ~30K chars. Warn when approaching limits. |
| Workspace skills contain prompt injection | Medium | Mandatory `VerifyContent()` scan for workspace-level skills. Critical patterns block loading. |
| ClawHub API unavailable | Low | Graceful degradation — skill install fails with clear error. All other functionality works offline. |
| YAML parsing edge cases from other tools | Medium | Lenient parser with fallback. Warn but load when possible. |
| Agent doesn't have Bash access | Very Low | All 4 current drivers provide Bash/shell tools. Document requirement. |

## Architecture Decision Records

- [ADR-001: Prompt-Only Runtime](adrs/adr-001.md) — Skills use prompt-only execution; no subprocess or WASM runtimes
- [ADR-002: SKILL.md Native Format](adrs/adr-002.md) — AgentSkills specification as the single native format
- [ADR-003: System Prompt + CLI Access Pattern](adrs/adr-003.md) — Catalog via system prompt, content via `agh skill view` CLI
- [ADR-004: Four-Level Loading Hierarchy](adrs/adr-004.md) — Bundled → user → .agents → workspace with override semantics
