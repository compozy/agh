# TechSpec: Extension Registry — Remote Discovery, Install, and Update

## Executive Summary

AGH needs remote extension discovery and installation. Today, extensions only support local-path install (`agh extension install <path>`), and skills have a partially-integrated ClawHub marketplace client. This TechSpec introduces a `RegistrySource` interface in a new `internal/registry/` package that abstracts multiple registry backends (ClawHub for skills, GitHub for both skills and extensions, future AGH Registry). Both the `agh skill` and `agh extension` CLI namespaces gain remote `install`, `remove`, and `update` commands backed by this shared interface.

**Important scope constraint**: ClawHub (OpenClaw) is a skills-only registry — its API exposes `/skills/*` endpoints exclusively. Extensions use GitHub Releases as the primary remote source for alpha. A future AGH Registry can serve both types.

Key architectural decisions: multi-source `RegistrySource` interface (ADR-001), separate CLI namespaces for skills and extensions (ADR-002), tar.gz as universal distribution format (ADR-003), and reuse of the existing `extensions` SQLite table for remote install tracking (ADR-004).

Primary trade-off: introducing a new `internal/registry/` package adds to the package count, but avoids duplicating registry logic between skills and extensions while keeping each backend independently replaceable. The existing tar.gz extraction pipeline in `internal/cli/skill_marketplace.go` is refactored into the shared `Installer` — not rewritten from scratch.

## System Architecture

### Component Overview

```
┌──────────────────────────────────────────────────────────────┐
│                     internal/registry/                        │
│                                                                │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │                   RegistrySource (interface)             │  │
│  │  Name() → string                                         │  │
│  │  Capabilities() → SourceCaps                             │  │
│  │  Search(ctx, query, opts) → []Listing                    │  │
│  │  Info(ctx, slug) → *Detail                               │  │
│  │  Download(ctx, slug, opts) → *DownloadResult             │  │
│  │  Close() → error                                         │  │
│  └──────────────┬──────────────┬──────────────┬────────────┘  │
│                 │              │              │                │
│  ┌──────────────▼┐  ┌─────────▼──────┐  ┌───▼────────────┐  │
│  │   ClawHub      │  │    GitHub       │  │  AGH Registry  │  │
│  │   (skills only │  │  (Releases API, │  │  (future,      │  │
│  │    refactored) │  │   skills+ext)   │  │   skills+ext)  │  │
│  └────────────────┘  └────────────────┘  └────────────────┘  │
│                                                                │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │                MultiRegistry (aggregator)                │  │
│  │  Queries all enabled sources concurrently               │  │
│  │  Merges and deduplicates results                        │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │           Installer (domain-agnostic pipeline)           │  │
│  │  Download → LimitReader → Extract tar.gz → Validate     │  │
│  │  manifest presence → Verify content → Move to temp dir  │  │
│  │  (domain registration handled by CLI caller)            │  │
│  └─────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
         │                              │
         │ Search/Download              │ Install/Update
         ▼                              ▼
┌────────────────────┐   ┌────────────────────────────────────┐
│  agh skill ...     │   │  agh extension ...                  │
│  search/install/   │   │  search/install/remove/update       │
│  remove/update     │   │                                     │
└────────────────────┘   └────────────────────────────────────┘
         │                              │
         ▼                              ▼
┌────────────────────┐   ┌────────────────────────────────────┐
│  internal/skills/  │   │  internal/extension/                │
│  Registry          │   │  Registry (SQLite)                  │
│  (.agh-meta.json   │   │  (extensions table)                 │
│   provenance)      │   │                                     │
└────────────────────┘   └────────────────────────────────────┘
```

**Data flow:**

1. User runs `agh extension search <query>` or `agh skill search <query>`
2. CLI creates a `MultiRegistry` from configured sources
3. `MultiRegistry` concurrently queries each enabled `RegistrySource`
4. Results are merged with priority-based dedup (later sources override earlier, following the existing `overlaySkill()` pattern in `internal/skills/registry.go:502-515`). When the same slug exists in multiple sources, the highest-priority source wins. If ambiguous, CLI requires `--from <source>`.
5. On `install <slug>`, the `Installer` downloads the tar.gz, wraps the compressed stream in `io.LimitReader` (default 50MB), tracks decompressed bytes with a counting writer (cap: 500MB) and file count (cap: 10000), extracts, validates manifest presence, and runs `VerifyContent()`. Domain-specific registration (provenance sidecar for skills, SQLite insert for extensions) is handled by the CLI caller — NOT by the Installer — to keep `internal/registry/` free of imports from `internal/extension/` or `internal/skills/`.
6. For extensions: metadata persisted in SQLite `extensions` table with `source = "marketplace"`; `SourceMarketplace` imposes a restricted security ceiling (see `internal/extension/capability.go:51-58`) — only `memory.read`, `observe.read`, `session.read`, `skills.read`, `tool.read` are allowed
7. For skills: metadata persisted as `.agh-meta.json` provenance sidecar (existing pattern in `internal/skills/provenance.go:17`)
8. If daemon is offline: install proceeds locally, daemon discovers on next boot via `Registry.List()`. If daemon is running: **Phase 1 does NOT call `Manager.Reload()`** — the existing `Reload()` at `manager.go:547-559` stops ALL extensions then restarts, which is a disproportionate blast radius for adding one extension. Instead, the CLI logs a message: "Extension installed. Restart the daemon to activate." A per-extension `Manager.LoadNew(name)` method is deferred to Phase 2 (see Known Risks).

## Implementation Design

### Core Interfaces

```go
// internal/registry/source.go

// SourceCaps declares which operations a registry backend supports.
// Follows the project pattern of discrete capability checks (no interface embedding).
type SourceCaps struct {
    Search bool // false = Search() returns ErrNotSupported
}

// RegistrySource abstracts one registry backend.
// Implementations must be safe for concurrent use.
type RegistrySource interface {
    Name() string
    Capabilities() SourceCaps
    Search(ctx context.Context, query string, opts SearchOpts) ([]Listing, error)
    Info(ctx context.Context, slug string) (*Detail, error)
    Download(ctx context.Context, slug string, opts DownloadOpts) (*DownloadResult, error)
    Close() error
}

// ErrNotSupported is returned when a source does not support an operation.
// Follows the project sentinel+typed error pattern (see extension/registry.go:23-30).
var ErrNotSupported = errors.New("registry: operation not supported")
```

Design notes on the interface:
- **`Download` takes `DownloadOpts` struct** (not bare `version string`) to carry the version AND the `--asset` flag for GitHub multi-asset releases. This follows the project convention of struct parameters (e.g., `SearchOpts`, `StartOpts` in `internal/acp/types.go:45-56`), not functional options.
- **`Capabilities()` returns `SourceCaps`** — declares which operations the backend supports. GitHub returns `SourceCaps{Search: false}` because the GitHub Releases API has no cross-repo search endpoint. `MultiRegistry.Search()` skips sources where `Capabilities().Search == false` instead of silently receiving empty results. This follows the project pattern of discrete capability checks (no interface embedding).
- **`Download` returns `*DownloadResult`** (not raw `io.ReadCloser`) to carry metadata from the server — content length for progress bars, resolved version, and optional server-side checksum. This matches the existing `marketplace.SkillArchive` pattern (`internal/skills/marketplace/types.go:15-20`) which returns `Slug`, `Version`, and `Data` together.
- **`Close()` for lifecycle** — HTTP clients hold connection pools; `Close()` ensures proper cleanup.
- **No `CheckUpdate` method** — update checks are implemented as a convenience on `MultiRegistry` using `Info()` + local version comparison via the existing `versionIsNewer()` function from `internal/cli/skill_marketplace.go:590-607`. This avoids every backend reimplementing identical version comparison logic.
- **`MultiRegistry` does NOT implement `RegistrySource`** — it is a separate aggregator type, so callers can distinguish between "query one source" and "query all sources."

```go
// internal/registry/multi.go

// MultiRegistry aggregates multiple RegistrySource implementations.
// It does NOT implement RegistrySource — it is a higher-level aggregator.
type MultiRegistry struct {
    sources []RegistrySource // ordered by priority (lowest first)
    logger  *slog.Logger
}

func NewMultiRegistry(logger *slog.Logger, sources ...RegistrySource) *MultiRegistry
func (m *MultiRegistry) Search(ctx context.Context, query string, opts SearchOpts) ([]Listing, error)
// Search skips sources where Capabilities().Search == false, logs skip at Debug level.
func (m *MultiRegistry) Info(ctx context.Context, slug string) (*Detail, error)
func (m *MultiRegistry) Download(ctx context.Context, slug string, opts DownloadOpts) (*DownloadResult, error)
func (m *MultiRegistry) CheckUpdate(ctx context.Context, slug, currentVersion string) (*UpdateInfo, error)
func (m *MultiRegistry) Close() error
```

```go
// internal/registry/installer.go

// Downloader abstracts the download capability so Installer
// can be tested without a full MultiRegistry.
type Downloader interface {
    Download(ctx context.Context, slug string, opts DownloadOpts) (*DownloadResult, error)
}

// Installer handles the download → extract → verify pipeline ONLY.
// It is domain-agnostic: it extracts the archive and validates that a manifest
// file exists (extension.toml or SKILL.md), but does NOT perform domain-specific
// registration. Provenance sidecar writing (skills) and SQLite insert (extensions)
// are the responsibility of the CLI caller. This prevents internal/registry/ from
// importing internal/extension/ or internal/skills/.
type Installer struct {
    downloader         Downloader
    maxArchiveSize     int64 // default 50MB (compressed stream limit)
    maxDecompressedSize int64 // default 500MB (total extracted bytes)
    maxFileCount       int   // default 10000 (per-archive file cap)
}

func NewInstaller(dl Downloader, opts ...InstallerOption) *Installer
func (i *Installer) Install(ctx context.Context, slug string, dlOpts DownloadOpts, targetDir string) (*InstallResult, error)
```

### Data Models

**Listing** — search result from any registry source:

```go
// internal/registry/types.go

type Listing struct {
    Slug        string `json:"slug"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Author      string `json:"author"`
    Version     string `json:"version"`
    Downloads   int    `json:"downloads"`
    Source      string `json:"source"` // registry name
    Type        string `json:"type"`   // "skill" or "extension"
}
```

**Detail** — full info for a single package:

```go
type Detail struct {
    Listing
    Readme     string   `json:"readme"`
    MCPServers []string `json:"mcp_servers,omitempty"` // preserves existing SkillDetail field
    Tags       []string `json:"tags"`
    License    string   `json:"license"`
    Repository string   `json:"repository"`
    Versions   []string `json:"versions"`
}
```

**DownloadOpts** — download parameters (struct, following project convention for opts):

```go
type DownloadOpts struct {
    Version string // target version; empty = latest
    Asset   string // specific asset name for multi-asset releases (GitHub --asset flag)
}
```

**DownloadResult** — structured download response (replaces raw `io.ReadCloser`):

```go
type DownloadResult struct {
    Reader      io.ReadCloser
    Slug        string // resolved slug
    Version     string // version resolved by server (from headers/API)
    ContentSize int64  // -1 if unknown; enables progress bars
    Checksum    string // optional server-side checksum for MITM detection
    ContentType string // e.g., "application/gzip"
}
```

**UpdateInfo** — result of an update check (computed by `MultiRegistry`, not by backends):

```go
type UpdateInfo struct {
    Slug           string `json:"slug"`
    CurrentVersion string `json:"current_version"`
    LatestVersion  string `json:"latest_version"`
    HasUpdate      bool   `json:"has_update"`
    Source         string `json:"source"` // which registry has the update
}
```

**InstallResult** — outcome of an install operation:

```go
type InstallResult struct {
    Slug        string `json:"slug"`
    Name        string `json:"name"`
    Version     string `json:"version"`
    Source      string `json:"source"`
    InstallPath string `json:"install_path"`
    Checksum    string `json:"checksum"`
}
```

**SearchOpts** — search parameters:

```go
// PackageType distinguishes skill from extension in search.
type PackageType string

const (
    PackageTypeSkill     PackageType = "skill"
    PackageTypeExtension PackageType = "extension"
    PackageTypeAll       PackageType = "" // match both
)

type SearchOpts struct {
    Limit  int         `json:"limit"`
    Offset int         `json:"offset"`
    Type   PackageType `json:"type"`
}
```

**ExtensionInfo additions** (to existing struct in `internal/extension/registry.go`):

```go
// Add to existing ExtensionInfo struct:
RegistrySlug    string // e.g., "compozy/oauth-bridge"
RegistryName    string // e.g., "clawhub", "github"
RemoteVersion   string // version from registry
```

**SQLite schema additions** (to existing `extensions` table):

```sql
ALTER TABLE extensions ADD COLUMN registry_slug TEXT;
ALTER TABLE extensions ADD COLUMN registry_name TEXT;
ALTER TABLE extensions ADD COLUMN remote_version TEXT;
```

### API Endpoints

No new HTTP API endpoints. Registry operations are CLI-only. The daemon does not proxy registry queries — the CLI queries registries directly.

**New CLI Commands — Extensions:**

| Command | Args | Description |
|---------|------|-------------|
| `agh extension search <query>` | `[--from <source>] [--limit N]` | Search remote registries for extensions |
| `agh extension install <slug>` | `[--from <source>] [--version <ver>]` | Download and install extension from registry |
| `agh extension remove <name>` | — | Uninstall extension: delete directory from disk + remove DB record via `Registry.Uninstall()`. Note: the existing `Uninstall()` at `internal/extension/registry.go:84-101` is DB-only — the CLI `remove` handler must also `os.RemoveAll()` the extension directory before calling `Uninstall()`, with rollback if DB deletion fails. |
| `agh extension update [name]` | `[--all] [--check]` | Update one or all remote-installed extensions |

**Updated CLI Commands — Skills** (already exist, wire to new `RegistrySource`):

| Command | Change |
|---------|--------|
| `agh skill search <query>` | Refactor to use `MultiRegistry` instead of direct ClawHub client |
| `agh skill install <slug>` | Refactor to use `Installer` pipeline |
| `agh skill remove <name>` | No change (already works) |
| `agh skill update [name]` | Refactor to use `MultiRegistry.CheckUpdate()`. Reuse existing `versionIsNewer()` from `skill_marketplace.go:590-607` (moved to shared package). |

## Integration Points

### ClawHub (OpenClaw Ecosystem)

- **Purpose**: Primary skill registry, maintaining parity with OpenClaw ecosystem. **Skills only** — ClawHub has no extension endpoints.
- **Implementation**: Refactor existing `internal/skills/marketplace/clawhub/Client` (which implements `marketplace.Registry` at `clawhub/client.go:34`) to implement `RegistrySource`
- **Base URL**: Configurable via `[skills.marketplace]` config (default: `https://clawhub.ai/api/v1`)
- **Endpoints used**: `GET /skills/search?q=<query>&limit=<N>&offset=<N>`, `GET /skills/<slug>` (info), `GET /skills/<slug>/download` (archive, returns `X-Skill-Version` header)
- **Migration note**: The existing `marketplace.Registry` interface (`internal/skills/marketplace/registry.go:7-12`) has `Download(ctx, slug) → *SkillArchive` (no version param). The ClawHub adapter must map the `RegistrySource.Download(ctx, slug, version)` signature — if `version` is empty, use `/skills/<slug>/download` (latest); if specified, use `/skills/<slug>/versions/<version>/archive` if ClawHub supports it, otherwise ignore and log a warning.
- **Error handling**: Existing exponential backoff (1s initial, 30s max, 3 retries) at `clawhub/client.go:18-20`

### GitHub Releases

- **Purpose**: Primary source for extensions, alternative source for skills. Supports both artifact types.
- **Implementation**: New `RegistrySource` using GitHub Releases API (REST, no SDK dependency)
- **Capabilities**: `SourceCaps{Search: false}` — GitHub is a **slug-only source**. The GitHub Releases API has no cross-repo "search releases" endpoint. `Search()` returns `ErrNotSupported`. Users must know the `owner/repo` slug. `MultiRegistry.Search()` skips this source automatically via `Capabilities()` check.
- **Authentication**: Optional `GITHUB_TOKEN` env var for private repos and rate limits. Unauthenticated: 60 req/hour. Authenticated: 5000 req/hour.
- **Slug format**: `owner/repo` (e.g., `compozy/agh-oauth-bridge`)
- **GitHub API endpoints used**:
  - `GET /repos/{owner}/{repo}/releases/latest` — fetch latest release (for `Info` and `Download` with empty version)
  - `GET /repos/{owner}/{repo}/releases/tags/{tag}` — fetch specific version
  - Each release object has `assets[]` with `name`, `browser_download_url`, `content_type`, `size`
  - Pagination: releases are paginated at 30/page; `Info` only fetches page 1 for version listing
  - Pre-release and draft releases are excluded by default
- **Asset naming convention**: The release must contain exactly one `.tar.gz` asset with name matching `<repo>-<version>.tar.gz` (e.g., `agh-oauth-bridge-1.2.0.tar.gz`). If multiple `.tar.gz` assets exist, the installer fails with an explicit error asking the user to specify the asset name via `DownloadOpts.Asset` (CLI `--asset` flag). If no `.tar.gz` asset exists, the installer falls back to GitHub's auto-generated source archive (note: auto-generated archives contain a top-level `<repo>-<tag>/` directory that the extraction pipeline must handle by walking into the first subdirectory if the manifest is not at root).
- **Content-Type validation**: Verify response `Content-Type` is `application/gzip`, `application/x-gzip`, or `application/octet-stream` before extraction. A 200 response with `text/html` (redirect to login) fails fast with a clear message.
- **Download**: Fetch tar.gz asset from latest (or specified) release tag
- **Error handling**: Same retry strategy as ClawHub, plus GitHub rate-limit awareness (`X-RateLimit-Remaining` header → warn at <10, fail at 0 with message suggesting `GITHUB_TOKEN`)

### Extension Manager

- **Integration point**: `internal/extension/manager.go`
- **Phase 1 behavior**: No daemon notification on install. The existing `Manager.Reload()` (`manager.go:547-559`) calls `Stop()` then `Start()` — this restarts ALL extensions, not just the new one. The `startOne()` method at line 769 is private and takes a `*managedExtension`, not a name. There is no per-extension load method. The blast radius of full-reload to install one extension is unacceptable for Phase 1.
- **Phase 1 approach**: CLI installs locally (SQLite + filesystem) and prints: `"Extension installed. Restart the daemon to activate, or it will be discovered on next boot."` Daemon discovers new extension on next `Start()` via `Registry.List()`.
- **Phase 2 (deferred)**: Add `Manager.LoadNew(ctx, name)` that starts a single new extension without stopping existing ones. This requires making the extension discovery/launch path work for individual extensions. The new UDS RPC `LoadNewExtension` would call this method. Tracked as a follow-up task.

### Skills Registry

- **Integration point**: `internal/skills/registry.go`
- **Change**: After remote skill install, call `registry.RefreshGlobal()` to pick up new skill
- **Behavior**: Same as today — installed skills land in `~/.agh/skills/` and are discovered on next scan

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/registry/` | New | New package (~8 files: types, source interface, multi, installer, extract, version). No risk — isolated module. | Implement from scratch + refactor shared logic from `skill_marketplace.go` |
| `internal/registry/clawhub/` | New | ClawHub adapter implementing `RegistrySource`. Skills-only. Wraps existing client logic. | Refactor from `internal/skills/marketplace/clawhub/`. Must adapt `Download(ctx, slug) → *SkillArchive` to `Download(ctx, slug, version) → *DownloadResult` |
| `internal/registry/github/` | New | GitHub Releases adapter implementing `RegistrySource`. Skills + extensions. | Implement from scratch |
| `internal/extension/registry.go` | Modified | Add 3 nullable columns to `ExtensionInfo`, update Install/scan queries. Low risk — additive. Schema defined inline in `globaldb/global_db.go:92-102`, not a separate schema file. | Add fields, update SQL in `global_db.go` |
| `internal/cli/extension.go` | Modified | Add `search`, remote `install`, `remove`, `update` commands. Medium risk — new user-facing commands. Note: existing `Uninstall()` is DB-only; `remove` needs filesystem cleanup. | Add 4 new subcommands |
| `internal/cli/skill_marketplace.go` | Modified | Extract shared extraction logic (`extractMarketplaceArchive`, `pathWithinRoot`, `cleanArchiveEntryPath`, `versionIsNewer`, `moveInstalledSkillDir`) into `internal/registry/`. Medium risk — significant refactoring (~300 lines moving). | Extract to shared package, update callers |
| `internal/cli/skill_commands.go` | Modified | Refactor `search`, `install`, `update` to use `MultiRegistry`. Low risk — same behavior, different backend. | Swap client calls |
| `internal/skills/marketplace/` | Deprecated | Old `marketplace.Registry` interface and ClawHub client replaced by `internal/registry/`. | Remove after migration, verify no other consumers |
| `internal/config/config.go` | Modified | Add `[extensions.marketplace]` config section (consistent naming with `[skills.marketplace]`). Low risk — additive. Must include `Validate()` method following existing pattern at `config.go:582-611`. Add warning log when `base_url` uses `http://` instead of `https://` (existing `normalizeBaseURL` in `clawhub/client.go:278` accepts HTTP silently). | Add struct + TOML parsing + validation + TLS warning |
| `internal/api/udsapi/` | Deferred (Phase 2) | `LoadNewExtension` RPC endpoint deferred until `Manager.LoadNew()` per-extension reload is implemented. Phase 1 relies on daemon restart. | No action in Phase 1 |

## Testing Approach

### Unit Tests

- **RegistrySource contract tests** (`source_contract_test.go`): Define a contract test suite that any `RegistrySource` implementation must pass:
  - Search with results, empty results, network errors
  - Download success with metadata (version, content size, checksum)
  - Context cancellation mid-download (reader must be closed, no goroutine leak)
  - Invalid JSON response bodies (graceful error, not panic)
  - Timeout behavior (context deadline exceeded)
  - Large response handling (streaming, not buffering entire response in memory)
- **MultiRegistry** (`multi_test.go`): Concurrent source querying, priority-based deduplication (later source wins), partial failure (one source errors, others succeed), empty sources list, ambiguous slug detection.
- **Installer** (`installer_test.go`):
  - tar.gz extraction (refactored from existing `extractMarketplaceArchive` tests)
  - `io.LimitReader` enforcement — archive exceeding `maxArchiveSize` fails before extraction completes
  - Manifest validation (extension.toml / SKILL.md at root)
  - Symlink rejection — all symlinks in archives are rejected (matching existing behavior, not just outside-root)
  - Temp dir cleanup on failure and on interrupted download
  - Content-Type validation (reject HTML responses from redirects)
  - Atomic move with backup-on-replace (refactored from `moveInstalledSkillDir`)
  - Rollback on update failure: if download succeeds but install fails, previous version remains intact
  - Stale temp dir cleanup: orphaned `.agh-*-install-*` directories cleaned on next run
- **ClawHub adapter** (`clawhub/client_test.go`): Reuse existing tests, adapt `Download(ctx, slug) → *SkillArchive` to new `RegistrySource` interface. HTTP test server for all endpoints. Test `version` parameter mapping (empty → latest, specified → versioned endpoint).
- **GitHub adapter** (`github/client_test.go`):
  - Releases API response parsing (single release, multiple releases)
  - Rate limit handling (`X-RateLimit-Remaining: 0` → clear error with GITHUB_TOKEN suggestion)
  - Private repo without `GITHUB_TOKEN` → auth error
  - Repository with no releases → clear error
  - Release with no tar.gz asset → clear error
  - Release with multiple tar.gz assets → error requiring `--asset` flag
  - Content-Type validation (HTML login page response → fast fail)
  - Asset naming convention: `<repo>-<version>.tar.gz` pattern matching
- **Extension registry** (`registry_test.go`): Test new columns (registry_slug, registry_name, remote_version) round-trip through insert/query. Test concurrent install of same extension → one success, one `ErrExtensionExists`.
- **Version comparison** (`version_test.go`): Test `versionIsNewer()` (moved from `skill_marketplace.go`) with semver, prerelease, invalid versions.

Mock requirements:
- HTTP test server for ClawHub and GitHub API responses
- `*DownloadResult` wrapping in-memory tar.gz for download tests
- Temporary directories for extraction tests
- `Downloader` interface mock for `Installer` testing (no full `MultiRegistry` needed)

### Integration Tests

- **Full install flow**: Search → select → install → verify installed → list shows it → update check → update → verify version changed
- **Multi-source search**: Configure two sources, verify results merged correctly with priority-based dedup
- **Offline fallback**: Registry unreachable, CLI shows clear error, local operations unaffected
- **Extension lifecycle**: Install remote → daemon reload notification → enable → health check → disable → remove (filesystem + DB)
- **Concurrent install**: Two parallel `agh extension install` for the same slug → one succeeds, one gets clean error
- **Interrupted download recovery**: Kill download mid-stream → no orphan temp dirs remain → retry succeeds

Test data: fixture tar.gz archives with valid/invalid manifests in `testdata/` directories.

## Development Sequencing

### Build Order

Organized into 3 shippable PRs, each independently passing `make verify`.

#### PR 1: Foundation + Installer (Steps 1-5)

1. **Extract shared logic from `internal/cli/skill_marketplace.go`** — Move `extractMarketplaceArchive()`, `pathWithinRoot()`, `cleanArchiveEntryPath()`, `versionIsNewer()`, and `moveInstalledSkillDir()` (~300 lines) into `internal/registry/extract.go` and `internal/registry/version.go`. **Critical**: also add decompression-size limit (`maxDecompressedSize`, default 500MB via counting writer wrapping `io.Copy` at the current unprotected line 414) and file-count cap (`maxFileCount`, default 10000). Update `skill_marketplace.go` to call the new shared functions. Split into 1a (extract + add limits) and 1b (update integration tests in `skill_marketplace_integration_test.go`). Run `make verify` after each sub-step. **This is a prerequisite for all subsequent steps.**
2. **`internal/registry/types.go`** — Define all shared types (Listing, Detail, DownloadOpts, DownloadResult, SourceCaps, SearchOpts, PackageType, UpdateInfo, InstallResult, Downloader interface, `ErrNotSupported` sentinel). Depends on step 1.
3. **`internal/registry/source.go`** — Define `RegistrySource` interface with `Capabilities()`. Depends on step 2.
4. **`internal/registry/multi.go`** — `MultiRegistry` aggregator with concurrent query logic, priority-based dedup, `Capabilities()`-aware `Search()` (skips sources where `Search == false`), and `CheckUpdate()` convenience method using `Info()` + `versionIsNewer()`. Depends on steps 2, 3.
5. **`internal/registry/installer.go`** — Domain-agnostic pipeline: Download → `io.LimitReader(maxArchiveSize)` → extract with decompressed-size tracking + file-count cap → validate manifest presence (extension.toml or SKILL.md) → `VerifyContent()` → move to temp dir. Returns `InstallResult` with path and checksum. Does NOT write provenance sidecars or SQLite rows. Depends on steps 1 (extraction functions), 2 (types), 3 (`Downloader` interface). Does NOT depend on `MultiRegistry`.

#### PR 2: Adapters + Extension CLI (Steps 6-10)

6. **`internal/registry/clawhub/`** — Refactor existing ClawHub client to implement `RegistrySource`. Returns `SourceCaps{Search: true}`. Depends on step 3. Adapts `Download(ctx, slug) → *SkillArchive` to `Download(ctx, slug, opts) → *DownloadResult`. Maps `DownloadOpts.Version`: empty → `/skills/<slug>/download`, specified → versioned endpoint if available. `DownloadOpts.Asset` is ignored (ClawHub has single-asset downloads).
7. **`internal/registry/github/`** — GitHub Releases adapter implementing `RegistrySource`. Returns `SourceCaps{Search: false}`. Depends on step 3. `Search()` returns `ErrNotSupported`. `Info()` calls `GET /repos/{owner}/{repo}/releases/latest` + page 1 of releases for version list. `Download()` uses `DownloadOpts.Asset` for multi-asset disambiguation. Handles auto-generated source archives (walks into `<repo>-<tag>/` prefix). Includes Content-Type validation, rate-limit handling, pre-release/draft exclusion.
8. **`internal/extension/registry.go` + `internal/store/globaldb/global_db.go`** — Add `registry_slug`, `registry_name`, `remote_version` columns to `ExtensionInfo` and update schema string in `global_db.go:92-102`. No dependency on registry package — pure DB/struct changes.
9. **`internal/config/config.go`** — Add `[extensions.marketplace]` config section (consistent with `[skills.marketplace]`). Include `Validate()` method following the pattern at `config.go:582-611`. Add warning log when `base_url` uses `http://` scheme. No dependency on registry package.
10. **`internal/cli/extension.go`** — Add `search`, remote `install`, `remove`, `update` commands. `install` calls `Installer.Install()` then performs domain-specific registration (SQLite insert via `extension.Registry.Install()`). `remove` does `os.RemoveAll(dir)` + `Registry.Uninstall(name)`. No daemon notification in Phase 1 — prints restart message. Depends on steps 4, 5, 7, 8, 9.

#### PR 3: Skill Migration + Cleanup (Steps 11-12)

11. **`internal/cli/skill_commands.go`** — Refactor existing skill commands to use `MultiRegistry`. `install` calls `Installer.Install()` then writes `.agh-meta.json` provenance sidecar via `skills.WriteSidecar()` (the domain-specific step). Depends on steps 4, 5, 6.
12. **Remove `internal/skills/marketplace/`** — Delete deprecated package after full test coverage confirms the new path works. Depends on step 11. **Gate**: all existing skill marketplace tests (unit + integration in `skill_marketplace_integration_test.go`) must pass against the new `internal/registry/clawhub/` adapter with real HTTP test server before deletion.

### Technical Dependencies

- **No new external dependencies**: tar.gz extraction uses Go stdlib (`archive/tar`, `compress/gzip`). HTTP uses stdlib `net/http`. GitHub API uses REST (no SDK needed). Version comparison uses existing semver logic.
- **Existing dependencies preserved**: `gopkg.in/yaml.v3` (already in go.mod), `github.com/BurntSushi/toml` (already in go.mod).
- **ClawHub API availability**: Required for the ClawHub adapter but not for GitHub adapter or local operations.
- **Existing `marketplace.Registry` interface** (`internal/skills/marketplace/registry.go:7-12`): Must be fully replaced by `RegistrySource` — the migration path is: step 6 (adapter), step 11 (CLI swap), step 12 (delete old interface).
- **Deferred to Phase 2**: `Manager.LoadNew()` per-extension reload, `LoadNewExtension` UDS RPC endpoint, priority ordering configuration UI.

## Monitoring and Observability

All logging uses `log/slog` per project conventions:

| Event | Level | Structured Fields |
|-------|-------|-------------------|
| Registry search | Debug | `source`, `query`, `result_count`, `duration_ms` |
| Registry download started | Info | `source`, `slug`, `version` |
| Registry download completed | Info | `source`, `slug`, `version`, `size_bytes`, `duration_ms` |
| Registry download failed | Error | `source`, `slug`, `version`, `error`, `retry_count` |
| Archive extraction | Debug | `slug`, `file_count`, `total_size_bytes` |
| Archive size limit exceeded | Warn | `slug`, `size_bytes`, `max_size_bytes` |
| Install completed | Info | `slug`, `version`, `source`, `install_path`, `checksum` |
| Install failed | Error | `slug`, `version`, `source`, `error`, `stage` |
| Update check | Debug | `slug`, `current_version`, `latest_version`, `has_update` |
| Security verification failed | Warn | `slug`, `warning_count`, `critical_count` |

## Technical Considerations

### Key Decisions

See Architecture Decision Records below for full details.

- **Multi-source RegistrySource interface** (ADR-001): Abstracts registry backends behind a Go interface. ClawHub, GitHub, and future registries implement independently.
- **Separate CLI namespaces** (ADR-002): `agh skill` and `agh extension` remain distinct. Users know what they're installing.
- **tar.gz universal format** (ADR-003): Single extraction pipeline for all sources. No git dependency.
- **Reuse extensions table** (ADR-004): Three nullable columns added — no new tables, no FK complexity.

### Known Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| ClawHub API changes break compatibility | Low | Adapter is isolated — changes contained to `internal/registry/clawhub/` |
| GitHub rate limiting blocks installs | Medium | Check `X-RateLimit-Remaining` header, warn at <10, fail at 0 with clear message suggesting `GITHUB_TOKEN`. Multiple CLI processes on shared machines can exhaust unauthenticated limits (60 req/hr). |
| Malicious tar.gz with path traversal | Medium | Reuse existing `pathWithinRoot()` and `cleanArchiveEntryPath()` from `skill_marketplace.go`. Reject ALL symlinks in archives (not just outside-root), matching existing extraction behavior. |
| Gzip decompression bomb | Medium | The current `io.Copy` at `skill_marketplace.go:414` is unbounded after decompression. A 10-byte gzip header can decompress to 10GB and exhaust disk, taking down SQLite databases. Fix: counting writer wrapping `io.Copy` with `maxDecompressedSize` (500MB default) + `maxFileCount` (10000 default). **This is shipped in PR 1, step 1 — before any new download paths exist.** |
| Large compressed archives exhaust disk | Low | Wrap download stream in `io.LimitReader(reader, maxArchiveSize)` BEFORE passing to extraction function. Default 50MB, configurable. Pattern exists at `clawhub/client.go:315` for error bodies. |
| Concurrent installs of same extension | Low | SQLite UNIQUE constraint on `name` column prevents duplicates. Test verifies one clean `ErrExtensionExists` error. |
| Stale versions after registry source goes offline | Low | `--check` flag compares local vs remote version, fails gracefully if unreachable. AGH is local-first — all local operations work offline. |
| HTTP redirect to login page disguised as download | Medium | Validate `Content-Type` header before extraction. Reject `text/html` responses with clear "authentication required" message. |
| Marketplace-sourced extensions get excessive capabilities | Medium | `SourceMarketplace` trust level imposes security ceiling (`capability.go:51-58`): only `memory.read`, `observe.read`, `session.read`, `skills.read`, `tool.read`. Document this in CLI install output. |
| No cryptographic signature verification | Accepted | Alpha limitation. Checksum verifies integrity (not authenticity). Document as future work — plan for GPG/sigstore signing when AGH Registry is built. |
| GitHub release with ambiguous assets | Medium | If release has multiple `.tar.gz` assets, fail with explicit error listing assets and suggesting `--asset <name>` flag. `DownloadOpts.Asset` carries this through the interface. |
| GitHub auto-generated archives have prefix dir | Medium | GitHub source archives contain `<repo>-<tag>/` top-level directory. Extraction pipeline must detect and walk into single-child root dirs when manifest is not at archive root. |
| `Manager.Reload()` stops all extensions | High | Existing `Reload()` calls `Stop()` then `Start()` — if `Start()` fails, ALL extensions are down. Phase 1 avoids this entirely by not calling Reload. Phase 2 adds `Manager.LoadNew()` for per-extension activation. |
| HTTP base_url accepted silently | Low | `normalizeBaseURL` in `clawhub/client.go:278` accepts HTTP URLs. Config `Validate()` allows both HTTP and HTTPS (`config.go:597`). Phase 1 adds warning log when HTTP is used. Phase 2 can enforce HTTPS-only. |
| Orphaned temp dirs after crash | Low | Temp dirs use deterministic prefix (`.agh-install-*`). Installer cleans stale dirs (>1 hour old) on startup. |
| GitHub Search not supported | Accepted | GitHub adapter returns `ErrNotSupported` for `Search()`. `MultiRegistry` skips it via `Capabilities()` check. Users must know the `owner/repo` slug. Documented in CLI help text. |

## Architecture Decision Records

- [ADR-001: Multi-Source RegistrySource Interface](adrs/adr-001.md) — Abstracts registry backends behind a Go interface for ClawHub, GitHub, and future sources
- [ADR-002: Separate CLI Namespaces for Skills and Extensions](adrs/adr-002.md) — Maintain distinct `agh skill` and `agh extension` command trees
- [ADR-003: tar.gz Archive as Universal Distribution Format](adrs/adr-003.md) — Single extraction pipeline using tar.gz for all registry sources
- [ADR-004: Reuse Existing SQLite extensions Table](adrs/adr-004.md) — Add three nullable columns for remote install tracking instead of new tables
