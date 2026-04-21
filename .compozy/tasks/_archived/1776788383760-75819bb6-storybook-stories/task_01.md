---
status: completed
title: Bootstrap dual Storybook infrastructure
type: infra
complexity: high
dependencies: []
---

# Task 1: Bootstrap dual Storybook infrastructure

## Overview
Stand up the two Storybook 10 instances that the rest of the rollout consumes: a fresh instance inside `packages/ui/` scoped to the `@agh/ui` primitives, and an extension of the existing `web/.storybook/` configuration that enables MSW plus global TanStack Query and memory-history router decorators. This foundation must render cleanly for both workspaces before any story authoring begins.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add a new Storybook 10 instance under `packages/ui/.storybook/` with `main.ts` glob `"../src/**/*.stories.@(ts|tsx)"` and dev port 6007.
- MUST import the shared design tokens via `@agh/ui/tokens.css` inside the new `preview.ts`; the packages/ui instance MUST NOT pull in `web/src/styles.css`, MSW, QueryClient, or router providers.
- MUST add `msw@^2` and `msw-storybook-addon@^2` as devDependencies of the `web` workspace and run `msw init web/public --save` once, committing the generated worker file.
- MUST extend `web/.storybook/preview.ts` with: `initialize({ onUnhandledRequest: "bypass" })`, `loaders: [mswLoader]`, a fresh-per-story `QueryClientProvider` (retry disabled, `staleTime: Infinity`), and a memory-history router decorator stub that does not require real route definitions.
- MUST keep the existing `withThemeByClassName` decorator and addon list in the web instance working without regressions.
- MUST add `storybook` and `build-storybook` scripts to `packages/ui/package.json` matching the web workspace versions.
- MUST NOT modify `packages/ui/src/` source code or introduce new runtime dependencies on `web/`.
</requirements>

## Subtasks
- [x] 1.1 Create `packages/ui/.storybook/main.ts` and `preview.ts` for a Storybook 10 + Vite instance covering only `packages/ui/src/**/*.stories.@(ts|tsx)`.
- [x] 1.2 Install Storybook 10 and required addons as devDependencies in the `packages/ui` workspace; wire `storybook`/`build-storybook` scripts.
- [x] 1.3 Add `msw` and `msw-storybook-addon` to `web/package.json` devDependencies; run the MSW service-worker init once and commit `web/public/mockServiceWorker.js`.
- [x] 1.4 Update `web/.storybook/preview.ts` to register the MSW loader, a story-scoped `QueryClientProvider`, and a memory-history router decorator while preserving the existing theme decorator.
- [x] 1.5 Validate both instances start (`bun run --cwd packages/ui storybook`, `bun run --cwd web storybook`) and that the existing `design-system/*` stories continue to render.

## Implementation Details
Follow the "System Architecture" and "Core Interfaces" sections of the TechSpec. The web preview fragment and packages/ui layout are described there; do not duplicate here. Ensure the two instances share only `@agh/ui/tokens.css` — the web instance additionally imports `web/src/styles.css` as today.

### Relevant Files
- `web/.storybook/main.ts` — existing config to leave glob untouched.
- `web/.storybook/preview.ts` — extend with MSW + providers.
- `web/package.json` — add `msw`, `msw-storybook-addon`.
- `web/public/` — target for generated `mockServiceWorker.js`.
- `packages/ui/package.json` — add Storybook devDeps + scripts.
- `packages/ui/src/tokens.css` — referenced by the new preview.
- `.compozy/tasks/storybook-stories/_techspec.md` — authoritative design.

### Dependent Files
- All downstream story tasks (`task_02`..`task_10`) consume these configs.
- `.claude/skills/storybook-stories/SKILL.md` — will be updated in `task_11` to match.

### Related ADRs
- [ADR-001: Dual Storybook Topology](adrs/adr-001.md) — Mandates the dual-instance setup delivered here.
- [ADR-002: MSW + Shared Decorators for System Stories](adrs/adr-002.md) — Drives the web-preview extension.

## Deliverables
- `packages/ui/.storybook/{main.ts,preview.ts}` and updated `packages/ui/package.json` scripts/devDeps.
- Updated `web/.storybook/preview.ts` with MSW + providers; committed `web/public/mockServiceWorker.js`.
- `web/package.json` updated with MSW devDeps.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for Storybook bootstrap **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `preview.ts` exports a decorator list that includes exactly one `QueryClientProvider`, one router stub, and preserves `withThemeByClassName`.
  - [x] The web `QueryClient` factory returns a client configured with `retry: false` and `staleTime: Infinity`.
  - [x] The packages/ui `preview.ts` imports `@agh/ui/tokens.css` and does not reference MSW or QueryClient.
- Integration tests:
  - [x] `bun run --cwd packages/ui build-storybook` exits 0 against an empty story set.
  - [x] `bun run --cwd web build-storybook` exits 0 with only the existing `design-system/*` stories present.
  - [x] Starting the web Storybook locally registers the MSW worker (`[MSW]` console log) and bypasses unknown requests.
- Test coverage target: >=80%
- All tests must pass

## Verification Notes

- `bunx vitest run src/storybook/web-storybook-config.test.tsx src/storybook/packages-ui-storybook-config.test.ts`
- `bunx vitest run --coverage.enabled --coverage.provider=v8 --coverage.reporter=text src/storybook/web-storybook-config.test.tsx src/storybook/packages-ui-storybook-config.test.ts`
- `bun run --cwd packages/ui build-storybook`
- `bun run --cwd web build-storybook`
- `bunx playwright test e2e/storybook-bootstrap.spec.ts`
- `make verify`

## Success Criteria
- All tests passing
- Test coverage >=80%
- Both Storybook instances build green on CI.
- Existing `design-system/*` stories render identically to pre-change.
- `mockServiceWorker.js` present under `web/public/` and referenced from `web/.storybook/preview.ts` via the MSW loader.
