# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- `task_01` persistence primitives are present in `internal/config` and verified in this run; later settings tasks can build on them instead of adding new persistence stores.

## Shared Decisions
- Canonical settings persistence stays file-backed in `internal/config` via semantic write targets instead of exposing absolute paths to higher layers.
- All config and MCP sidecar writes must validate the merged effective config before disk commit.

## Shared Learnings
- `internal/config` now exposes reusable write helpers around `WriteTarget` resolution, `EditConfigOverlay`, and MCP sidecar put/delete helpers; task_02 can orchestrate section and collection writes through those primitives.
- MCP sidecar writes preserve unknown top-level JSON keys and untouched server definitions while TOML overlay writes reject unsupported structural mutations instead of whole-file canonicalization.
- Restart orchestration now persists durable operation records under `HomePaths.RestartsDir`; later settings API/handler tasks should consume `internal/daemon` restart methods instead of reading restart JSON files directly.
- Replacement-daemon readiness is only valid after fresh daemon discovery state exists; transports should treat persisted `stopping`, `waiting_release`, and `starting` states as in-progress and surface terminal `failed` reasons from the stored record.
- `internal/api/core` now owns the shared settings parsing, DTO conversion, error mapping, restart trigger/status handling, and observability log-tail SSE plumbing; later transport tasks should register routes and transport policy only, not reimplement handler behavior.
- `internal/daemon` now wires `internal/settings.Service` plus a restart-controller adapter during boot, and both HTTP/UDS servers accept injected settings dependencies through shared constructor options.
- HTTP now registers the full `/api/settings/*` namespace plus `/api/extensions`; privileged HTTP mutations are guarded at route middleware based on the configured bind host, while read-only settings routes and restart-status polling remain available on non-loopback binds.
- UDS now mirrors the full `/api/settings/*` namespace without the HTTP loopback mutation guard, and `internal/api/udsapi` carries unit plus integration parity coverage for settings and extension surfaces, including non-loopback HTTP vs privileged UDS workspace-scoped `mcp-servers` mutations.
- `web/src/systems/settings` is the canonical frontend domain for every section, collection, restart action, and log-tail URL; section pages must consume it via the public barrel and route-level orchestration should live in `web/src/hooks/routes/use-settings-page.ts` instead of per-route fetch logic.
- Restart polling state is centralized in `settings-restart-store` (Zustand) and exposed through `useSettingsRestart()`; polling stops as soon as status reaches `ready` or `failed`, and `useSettingsPage().restart` projects the combined banner state for any settings route.
- Shared section UI primitives live under `web/src/systems/settings/components/` (`SettingsPageShell`, `SettingsSectionCard`, `SettingsFieldRow`, `SettingsStatusLine`, `SettingsSaveBar`, `SettingsRestartBanner`) and must be reused by every settings page instead of building per-section shells.
- Route-level orchestration for settings pages belongs in `web/src/hooks/routes/use-settings-<slug>-page.ts` (pattern established by task_10 for general/memory/observability) so the route file stays presentational and passes `compozy-react/max-component-complexity`.
- Memory "Trigger now" consolidate reuses `useConsolidateMemory` from `@/systems/knowledge` — do not introduce a new settings-owned consolidate adapter.

## Open Risks

## Handoffs
- `task_02` should consume the semantic target kinds and persistence helpers already in `internal/config` rather than duplicating file-path logic.
- `task_04` and `task_05` should expose the persisted restart operation through contract/core using the daemon-owned status model rather than inventing a transport-specific restart state machine.
- `task_06` and `task_07` should reuse the shared `api/core` settings handlers and keep HTTP loopback enforcement or UDS-specific policy in transport wiring only.
- `task_09`+ can import `SETTINGS_SECTIONS` from `web/src/routes/_app/settings.tsx` (or relocate it into `systems/settings/lib`); per-section pages just need `web/src/routes/_app/settings/<slug>.tsx` files and the shell automatically frames them.
- task_10..task_14 should build section pages on top of `@/systems/settings` hooks (reads + mutation hooks) and reuse `useSettingsPage` in the shell — route files must stay presentational with no direct `/api/settings/*` calls.
- task_11+ section pages must reuse `@/systems/settings/components` primitives and follow the `use-settings-<slug>-page.ts` orchestration hook pattern introduced in task_10; do not re-implement page shells, save bars, or restart banners per section.
