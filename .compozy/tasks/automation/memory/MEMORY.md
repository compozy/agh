# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 completed: automation config loading/validation now exists in `internal/config`, and the canonical automation model now exists in `internal/automation`.
- Task 06 completed: the daemon now boots a built-in automation manager after extensions and before servers, publishes it through runtime dependencies, and shuts it down deterministically before session shutdown.
- Task 09 completed: extensions can now read/manage automation through Host API methods, emit `ext.*` trigger events through the shared trigger engine, and observe automation lifecycle hooks (`pre_fire`, `post_fire`, `run.completed`, `run.failed`) through the canonical hooks runtime.
- Task 10 completed: the SPA now exposes a unified `/automation` management surface with job/trigger list-detail flows, create/edit forms, manual job runs, sidebar navigation, and a dedicated `web/src/systems/automation` module.

## Shared Decisions
- Keep TOML-facing automation structs in `internal/config`; later persistence, runtime, transport, and API layers should map onto `internal/automation` instead of defining parallel models.
- `internal/store/globaldb` is now the authoritative automation persistence layer. Later runtime and API tasks should call its Go methods for jobs, triggers, runs, overlays, webhook lookup, and fire-limit queries instead of issuing SQL elsewhere.
- Enabled overlays are enforced as config-only operational state. Dynamic jobs and triggers must use direct definition updates; overlay rows are reserved for TOML-backed definitions.
- Runtime code in `internal/automation` cannot import `internal/config` transitively through `session`/`acp` without creating an import cycle. The pure automation model and validation layer now lives in `internal/automation/model`; `internal/config` and `internal/store/globaldb` should keep importing that leaf package while runtime code continues through `internal/automation` aliases.
- Global automation dispatch should create system sessions with `session.CreateOpts.WorkspacePath` pointed at the AGH home directory, while workspace-scoped automation should use `session.CreateOpts.Workspace = workspace_id`.
- The scheduler runtime lives under `internal/automation/schedule.go` as a thin wrapper over `gocron v2`. Later manager and API tasks should consume its `Register` / `Update` / `Unregister` / `State` / `States` surface for schedule lifecycle and next-run metadata instead of reaching into `gocron` directly.
- The trigger runtime now lives under `internal/automation/trigger.go` as an in-memory registry over the shared `Dispatcher`. Later manager/API work should register effective trigger definitions plus their write-only webhook secrets into that runtime instead of querying persistence on every activation.
- The daemon owns automation composition. Later API, CLI, UI, and extension tasks should consume `RuntimeDeps.Automation` / `core.AutomationManager` rather than constructing another manager or reaching into scheduler/trigger internals directly.
- The canonical automation daemon API now exists through shared `internal/api/core` handlers plus `/api/automation/*` management routes on HTTP and UDS. Webhook delivery stays HTTP-only under `/api/webhooks/*`, and the canonical OpenAPI document/generated web types now include the automation operations.
- Extension automation must stay on the daemon-owned runtime. Host API automation methods call the same manager used by the core API, and extension-provided `ext.*` events must enter through `automation/triggers/fire` so they reuse the trigger engine and dispatcher instead of bypassing runtime governance.
- Daemon integration uses hook/session fan-out adapters so automation can attach after hooks boot while preserving the existing extension startup path. Reuse those fan-outs for future built-in automation ingress instead of wiring direct subscriptions into `session.Manager` or `hooks.Runtime`.
- Dynamic webhook trigger secrets now persist in `globaldb` write-only storage (`automation_trigger_webhook_secrets`), and the manager's default runtime secret resolver reads from that store when registering webhook triggers.
- Dynamic webhook trigger creation may omit `webhook_id`; the manager now backfills a stable `wbh_...` identifier from the persisted trigger ID before runtime registration so API/CLI/UI callers can rely on the returned webhook endpoint shape.
- Web automation should stay inside the system-layer module under `web/src/systems/automation`; future UI work should extend its adapters/query hooks and reuse generated OpenAPI automation types instead of introducing route-local fetchers or parallel client DTOs.

## Shared Learnings
- Strict trigger prompt validation cannot rely on `text/template` plus `missingkey=error` alone because `index .Data "key"` does not fail on missing map keys; preserve the AST-based validation in `internal/automation/template.go`.
- Stable webhook resolution is keyed by persisted `webhook_id`, not `endpoint_slug`. Slugs may change without breaking lookup as long as the stable webhook identifier stays the same.
- Restart-safe fire-limit evaluation should use persisted run-window queries in `globaldb` (`CountRuns` / `ListRuns`) over `started_at` windows rather than any in-memory counters.
- Scheduler shutdown can cancel in-flight automation executions, but terminal run persistence must still succeed afterward. `internal/automation/dispatch.go` now uses a cancellation-free persistence context for final `UpdateRun` writes so cancelled runs are recorded correctly.
- One-shot `at` schedules are not backfilled when their target time is already in the past. The scheduler skips registration for those jobs and unregisters successful one-shot jobs after their first fire.
- Internal trigger ingress already has boundary-shaped adapters: session lifecycle uses `session.Notifier`, hook completions use `hooks.TelemetrySink`, and memory consolidation uses the observer-facing `MemoryConsolidationObserver`. Later daemon wiring should reuse those seams instead of adding new subscriptions inside `session` or `memory/consolidation`.
- Clean automation shutdown should happen after extensions stop and before session shutdown so scheduler-triggered prompts see context cancellation without leaving daemon-owned runtime goroutines behind.
- Hook catalog consumers should derive supported-event counts from `hookspkg.AllHookEvents()` / `AllEventDescriptors()` instead of hardcoding totals, because new hook families like automation lifecycle events are additive by design.

## Open Risks
- Config-defined jobs and triggers still carry pre-resolution workspace bindings in `internal/config`; later tasks must resolve them to canonical workspace IDs before persistence and runtime use.
- Webhook secrets remain intentionally absent from the readable trigger model. Dynamic triggers now persist write-only secrets for runtime registration, but config-defined webhook triggers still need a secure write-only source if they are expected to register automatically after startup.

## Handoffs
- Task 02 should build persistence around the shared `internal/automation` types and reuse config validation outputs instead of re-validating shape invariants from scratch.
- Task 03 can build dispatcher fire-limit checks on the new `CountRuns` / `ListRuns` queries and should keep overlay application explicit by reading definition rows separately from overlay rows.
- Task 04 and Task 05 should call the shared dispatcher from schedules and triggers rather than recreating their own session/run governance. The dispatcher already owns concurrency, retries, fire limits, and session prompt handoff.
- Task 06 should own daemon/manager composition around the new scheduler runtime and avoid adding separate background scheduling loops; `Scheduler.Start` and `Scheduler.Shutdown` are the intended lifecycle hooks.
- Task 07 and Task 10 can surface scheduled next-run data from `Scheduler.State` / `States` rather than recalculating schedule timestamps in transport or UI code.
- Task 08 CLI automation commands can treat HTTP/UDS automation CRUD and history routes as the canonical control surface; webhook delivery remains HTTP-only while management routes exist on both transports.
- Task 10 can consume the regenerated automation entries in `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` instead of defining ad hoc client types.
