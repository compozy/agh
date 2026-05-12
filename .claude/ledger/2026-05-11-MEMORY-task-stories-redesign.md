---
session: task-stories-redesign
created: 2026-05-11
plan: /Users/pedronauck/.claude/plans/n-s-usamos-o-prompt-composed-reef.md
---

# Goal

Apply the redesign proposals (`docs/design/new-proposal/redesign-*.html`) to the actual React components renderered by these 3 stories, reusing primitives from `@agh/ui`:

- `systems-tasks-taskoverviewcomponents--operational-cards`
- `systems-tasks-taskoverviewcomponents--orchestration`
- `systems-tasks-taskrundetail--running`

Wide polish pass approved; `/tasks/$id` route is out of scope.

# Constraints / Assumptions

- Greenfield alpha — no compat shims.
- Reuse `@agh/ui` primitives; do not duplicate `<Timeline>`, `<TimelineEvent>`, `<RunCard>`, `<DetailHeader>`, `<Section>`.
- Preserve all existing `data-testid` values used by `__tests__/*.test.tsx`.
- No new HTML files; no fixture changes; no `border-l-*` accent stripes (banned).

# Key decisions

- **Single structural delta**: `tasks-timeline-panel.tsx` migrates from custom `<ol>` + `<Pill.Dot>` to `<Timeline>` + `<TimelineEvent>` (matching `task-run-timeline-panel.tsx`).
- **DRY win**: extract `EVENT_VISUALS`, `visualFor`, event-type sets, and `describeEvent` to `web/src/systems/tasks/lib/timeline-visuals.ts`. Both panels consume it.
- **Section polish**: add `icon` + `count` to `<Section>` in 5 cards (reviews, bridge notifications, execution profile, stream resume, timeline panel). Move "Live" pulse to Section `right` slot in timeline panel.
- **Test-id preservation**: `tasks-timeline-event-type-${id}` and `tasks-timeline-message-${id}` must persist when `<TimelineEvent>` replaces the `<li>` — set them on the inner `title`/`description` spans.

# State

## Done

- Read all 6 component sources + 6 `__tests__/*.test.tsx` for impact analysis.
- Inventoried `@agh/ui` primitives; got plan approval.
- Captured baselines: `/tmp/agh-redesign-shots/before/{operational-cards,orchestration,taskrundetail-running}.png`.
- Created `web/src/systems/tasks/lib/timeline-visuals.ts` (EVENT_VISUALS, visualFor, event-type sets, describeEvent, resolveEventTone, isFailureEvent, isLiveEvent, isSuccessEvent).
- Refactored `task-run-timeline-panel.tsx` to consume helper (-98 lines local duplication).
- Migrated `tasks-timeline-panel.tsx` to `<Timeline>` + `<TimelineEvent>`; moved Live indicator to Section `right` slot; added `icon={Activity}` + `count={items.length}` to Section; preserved every `data-testid`.
- Added `icon`/`count` to Sections in `tasks-reviews-card` (Gavel + count), `tasks-bridge-notifications-card` (Radio + count), `tasks-execution-profile-card` (Settings2, no count), `tasks-stream-resume-card` (Activity, no count).
- Lint on modified files: 0 warnings, 0 errors (`bunx oxlint` on 7 files).
- Typecheck via Turbo: passes (`bunx turbo run typecheck --filter=./web`).
- Tests on the 6 touched components: 36/36 pass.
- Captured after screenshots: `/tmp/agh-redesign-shots/after/{operational-cards,orchestration,taskrundetail-running}.png`.

## Now

- Reporting to user.

## Pre-existing failures (NOT caused by this work)

- 34 tests fail in unrelated files (`task-run-detail-header.test.tsx`, `tasks-detail-header.test.tsx`, `tasks-multi-agent-panel.test.tsx`, `tasks-list-row.test.tsx`, `-tasks.$id.test.tsx`, `-bridges.test.tsx`, etc.).
- Root cause: `vi.mock("@tanstack/react-router")` missing `useRouter` export. Mock infrastructure issue in those test files, not in the components touched here.
- 2 lint warnings pre-exist in `src/systems/network/contexts/network-list-filters-context.tsx` (hooks-in-hooks-folder, no-mixed-hooks-and-components) — not touched by this work.

## Next

- (User decision) Whether to commit / open a follow-up to repair pre-existing mocks + lint warnings.

# Open questions

- None (scope decided via AskUserQuestion).

# Working set

- `web/src/systems/tasks/components/tasks-timeline-panel.tsx`
- `web/src/systems/tasks/components/task-run-timeline-panel.tsx`
- `web/src/systems/tasks/components/tasks-reviews-card.tsx`
- `web/src/systems/tasks/components/tasks-bridge-notifications-card.tsx`
- `web/src/systems/tasks/components/tasks-execution-profile-card.tsx`
- `web/src/systems/tasks/components/tasks-stream-resume-card.tsx`
- `web/src/systems/tasks/lib/timeline-visuals.ts` (NEW)
- `web/src/systems/tasks/components/__tests__/tasks-timeline-panel.test.tsx`
- Tests for the 5 other cards (read-only verification; should not need changes).

Verification commands:

- `make web-lint`
- `bunx turbo run typecheck --filter=./web`
- `bunx turbo run test --filter=./web`
- `agh-ui-screenshot` for 3 stories before+after
