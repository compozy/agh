# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 is implemented and verified. `internal/skills` now exposes marketplace/MCP/hook/provenance foundation types for later skills-v2 tasks.
- Task 03 is implemented and verified. Loader paths now populate `Skill.MCPServers` and `Skill.Hooks` from `metadata.agh` for both disk and bundled skills.
- Task 04 is implemented and verified. `internal/skills/mcp.go` now resolves skill MCP servers with trust-tier filtering, stable source-precedence deduplication, and structured logging.
- Task 06 is implemented and verified. `internal/skills/provenance.go` now provides sidecar hashing, read/write, detection, and tamper verification helpers for marketplace skills.
- Task 07 is implemented and verified. `internal/skills/registry.go` now promotes sidecar-backed global skills to `SourceMarketplace`, loads provenance/`InstalledFrom`, verifies hashes on every load, logs tamper mismatches, and blocks critically flagged marketplace skills; `internal/skills/catalog.go` now excludes disabled skills from prompt catalogs.
- Task 10 is implemented and verified. `internal/cli/skill.go` now exposes `search`, `install`, `remove`, and `update` marketplace subcommands, and `internal/config` now carries the marketplace registry/base URL config needed to construct the CLI client.
- Task 09 is implemented and verified. `internal/session.Manager` now merges skill-resolved MCP servers into agent start options during create/resume, and `internal/daemon.notifierFanout` now dispatches subprocess hooks after built-in notifiers using the resolved workspace context.

## Shared Decisions
- `SourceMarketplace` precedence sits between bundled and user skill sources.
- Registry clones deep-copy `MCPServers`, `Hooks`, and `Provenance` so later tasks can enrich runtime skill copies without mutating registry snapshots.
- MCP resolution sorts active skills by `SkillSource` before deduplicating by server name, so higher-precedence sources override lower-precedence ones even if the caller passes an unsorted skill slice.
- Session startup consumes a paired `SkillRegistry` + `MCPResolver` dependency; both must be configured together or omitted together so create/resume can resolve active workspace skills and merge MCP servers deterministically.
- Daemon hook dispatch is a dedicated post-notifier phase, not a `session.Notifier`; it resolves the session workspace through `WorkspaceResolver` before loading active skills and running `HookRunner`.

## Shared Learnings
- Marketplace workspace-path source strings are treated as global/non-overlay entries in `skillSourceFromWorkspacePath()`.
- `internal/config.SkillsConfig` now carries `AllowedMarketplaceMCP`, and config overlay parsing accepts `skills.allowed_marketplace_mcp` for resolver consent input.
- `internal/config.SkillsConfig` now also carries `Marketplace` (`registry`, `base_url`); validation permits the zero-value default but requires `skills.marketplace.registry` when marketplace settings are explicitly configured.
- Provenance verification operates on raw `SKILL.md` file bytes, and `VerifyHash` returns a `HashMismatchError` with expected and actual hashes so registry code can log tamper details without reparsing error strings.
- Registry snapshot invalidation for global skills now includes `.agh-meta.json` sidecars, so marketplace install/remove/provenance updates trigger `RefreshGlobal()` even when `SKILL.md` bytes are unchanged.
- Disabled skills remain present in registry results for management flows, but prompt catalog assembly now filters `Enabled == false` entries so disabled skills are not exposed to agents.
- The ClawHub client normalizes registry base URLs so callers can pass either a host root or a full `/api/v1` base without duplicating API path segments.
- The CLI marketplace flows default a blank configured registry to `clawhub`, block installs on critical `VerifyContent` findings, and reuse sidecar-backed installs for remove/update operations.

## Open Risks
- None currently recorded.

## Handoffs
- Future MCP/hook/provenance tasks can rely on typed metadata being available directly on loaded `Skill` values instead of reading `Meta.Metadata["agh"]` again.
- Task 07 can call `ReadSidecar`, `HasSidecar`, and `VerifyHash` from `internal/skills/provenance.go`; task 10 can reuse `ComputeHash` and `WriteSidecar` during marketplace installs/updates.
- Task 09 can construct `NewMCPResolver(cfg.Skills, logger)` directly from loaded config and merge `Resolve(activeSkills)` output into session start options.
- Future marketplace follow-up work can construct `clawhub.NewClient(baseURL)` directly from `cfg.Skills.Marketplace`; `SkillArchive.Data` is an `io.ReadCloser`, so install/update flows must close it after extraction.
