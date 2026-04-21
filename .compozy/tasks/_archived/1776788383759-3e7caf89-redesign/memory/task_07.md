# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Added `Metric`, `MonoBadge`, `KindChip`, `StatusDot`, `ConnectionIndicator` to `@agh/ui`.
- Migrated every importer of `web/src/components/design-system/{metric-strip,status-dot,connection-indicator}` + the showcase to the new primitives.
- Deleted the four legacy files (status-dot.tsx, metric-strip.tsx, connection-indicator.tsx, plus their two stories). The design-system folder now only holds `design-system-showcase.tsx` + `index.ts` + one story (task 15 clears the rest).

## Important Decisions

- Tone vocabulary moved to DESIGN.md §4 names: `success | warning | danger | info | accent | neutral`. Legacy `amber/green/violet` callers were mapped per intent (daemon offline → warning, running → accent, scope=workspace → info, etc.).
- `Metric` exposes two complementary slots: `detail` (inline baseline-aligned mono, per mock `primitives.jsx`) and `subtext` (Inter 13px secondary line, per DESIGN.md §4 "With Subtext"). Legacy `MetricStrip detail="…long sentence…"` call sites map to `subtext`.
- `ConnectionIndicator` default labels standardized to DESIGN.md wording: "Connected" / "Disconnected" / "Reconnecting" (dropped the "Offline" + ellipsis variants from legacy). Connection-status unit test updated in the same PR.
- `automation-run-history.tsx` switched from `automationSemanticTone` (old amber/green/violet) to `automationStatusTone` (new accent/success/warning/danger/neutral) — `automationSemanticTone` stays in place for `AutomationTag`, which still uses the legacy enum.

## Learnings

- `useReducedMotion()` from `motion/react` reads `matchMedia` directly and ignores a wrapping `MotionConfig`. When asserting pulse suppression against `<MotionConfig reducedMotion="…">` in vitest, components must call `useReducedMotionConfig()` (context-aware). Same trap flagged in shared workflow memory — landed it in `StatusDot`.

## Files / Surfaces

- `packages/ui/src/components/{status-dot,mono-badge,kind-chip,metric,connection-indicator}.tsx` (new) + matching `*.test.tsx` (new) + `stories/*.stories.tsx` (new).
- `packages/ui/src/index.ts` — added exports + types for the five primitives.
- `web/src/components/design-system/{status-dot,metric-strip,connection-indicator}.tsx` — deleted.
- `web/src/components/design-system/stories/{status-dot,metric-strip}.stories.tsx` — deleted.
- `web/src/components/design-system/{index.ts,design-system-showcase.tsx}` — rewired to `@agh/ui` Metric/StatusDot.
- Call-site migrations: `routes/_app/network.tsx`, `systems/daemon/{components/connection-status.tsx,hooks/use-daemon-health.ts}`, `systems/automation/components/automation-run-history.tsx`, `systems/settings/components/settings-status-line.tsx`, `systems/daemon/components/connection-status.test.tsx`.

## Errors / Corrections

- First StatusDot draft used plain `useReducedMotion()` → pulse test under `<MotionConfig reducedMotion="never">` failed because matchMedia still returned reduced in jsdom. Replaced with `useReducedMotionConfig()` and the suite went green.

## Ready for Next Run

- Task 15 can delete `web/src/components/design-system/` entirely once the showcase route is rewired or moved; the folder now only carries the showcase + its single story.
- Go `make verify` still blocked by the two pre-existing lint issues tracked in shared workflow memory (`internal/observe/tasks.go`, `internal/store/globaldb/global_db_task_aux.go`). Not in scope here.
