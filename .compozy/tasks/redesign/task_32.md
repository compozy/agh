---
status: pending
title: Rewrite Settings pages batch 2 (mcp-servers, hooks-extensions, observability, environments, network)
type: frontend
complexity: high
dependencies:
  - task_30
---

# Task 32: Rewrite Settings pages batch 2 (mcp-servers, hooks-extensions, observability, environments, network)

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite the five remaining Phase 6 sub-route pages `/settings/mcp-servers`, `/settings/hooks-extensions`, `/settings/observability`, `/settings/environments`, and `/settings/network` on top of the Settings shell from task 30 and the same `@agh/ui` primitive set used in task 31. Every page keeps its query hooks, mutations, and validation intact — only the visual layer is replaced.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `web/src/routes/_app/settings/mcp-servers.tsx`, `hooks-extensions.tsx`, `observability.tsx`, `environments.tsx`, and `network.tsx` using `@agh/ui` `Section`, `Field`, `Input`, `Switch`, `Pills`, `Combobox`, `NativeSelect`, `Table`, `Button`, `Dialog`, `Empty`, and the `SettingsPageShell` / `SettingsSaveBar` / `SettingsRestartBanner` from task 30.
- MUST preserve every existing hook call (`useSettingsMcpServersPage`, `useSettingsHooksExtensionsPage`, `useSettingsObservabilityPage`, `useSettingsEnvironmentsPage`, `useSettingsNetworkPage`) unchanged — presentational rewrite only.
- MUST preserve mutation payload shapes; PATCH / POST / DELETE requests keep their current JSON body so backend tests stay valid.
- MUST preserve validation, dirty detection, Save / Discard wiring, and restart-banner triggers (network + observability + environments are restart-sensitive).
- MUST NOT introduce any imports from `@/components/ui/*` or `@/components/design-system/*`; replace any `PillButton` usage with `@agh/ui` `Pills`.
- MUST render the MCP-servers list and the Environments list with `@agh/ui` `Table`, including a status `Pills` column for reachable / unreachable and a mono ID column per DESIGN.md §4.
- MUST render the Observability page metric grid through `SettingsStatGrid` (task 30 output) wrapping `@agh/ui` `Metric` primitives.
- MUST render empty states through `@agh/ui` `Empty` for `mcp-servers` (no servers configured), `hooks-extensions` (no hooks registered), and `environments` (no environments defined).
- MUST produce Playwright snapshot baselines per sub-route in at least the idle and dirty states; `mcp-servers`, `hooks-extensions`, and `environments` additionally snapshot the empty state.
- SHOULD keep file sizes manageable by extracting per-page section components inside the same route file when a page exceeds ~300 lines.
</requirements>

## Subtasks

> Subtasks below describe sub-routes, not individual files — each sub-route owns its route file plus any in-file section subcomponents. No new component files land in `web/src/systems/settings/components/`.

- [ ] 32.1 Rewrite `/settings/mcp-servers` — server `Table` (name, command, transport, status `Pills`), add-server `Dialog` with `Field` + `Combobox` for transport, per-row edit / delete actions, `Empty` state.
- [ ] 32.2 Rewrite `/settings/hooks-extensions` — hook groups (`Section` per hook kind), enabled `Switch`, inline script `Input`, add-hook `Dialog`, `Empty` state when no hooks are registered.
- [ ] 32.3 Rewrite `/settings/observability` — metric grid (sessions, events, rate) through `SettingsStatGrid` + `Metric`; log-level `NativeSelect`; OTLP endpoint `Field`; restart-banner wiring.
- [ ] 32.4 Rewrite `/settings/environments` — environment `Table` (name, scope, var count, default `Switch`), add / edit `Dialog` with variable editor, restart-banner wiring when defaults change, `Empty` state.
- [ ] 32.5 Rewrite `/settings/network` — bind-address / port `Field`s, TLS `Switch`, allowed-origins `Pills` input, CORS `NativeSelect`; restart-banner wiring on any submit.
- [ ] 32.6 Update the matching `*.stories.tsx` files under `web/src/routes/_app/settings/stories/` to consume the new components and exercise idle + dirty + loading + empty states.
- [ ] 32.7 Regenerate Playwright snapshot baselines per sub-route (idle + dirty minimum; add empty for `mcp-servers`, `hooks-extensions`, `environments`).

## Implementation Details

See TechSpec Impact Analysis row `web/src/routes/_app/settings/**` and ADR-004 Phase 6. These five sub-routes share the same primitive composition as task 31 — the split exists to keep per-PR review manageable, not because the primitives differ. DESIGN.md §4 "Cards & Containers" and "Inputs" define the visual rules; the Alert tint formula in DESIGN.md §2 governs the restart-banner tones each page feeds.

### Relevant Files

- `web/src/routes/_app/settings/mcp-servers.tsx` — rewrite target.
- `web/src/routes/_app/settings/hooks-extensions.tsx` — rewrite target.
- `web/src/routes/_app/settings/observability.tsx` — rewrite target.
- `web/src/routes/_app/settings/environments.tsx` — rewrite target.
- `web/src/routes/_app/settings/network.tsx` — rewrite target.
- `web/src/routes/_app/settings/stories/-mcp-servers.stories.tsx`, `-hooks-extensions.stories.tsx`, `-observability.stories.tsx`, `-environments.stories.tsx`, `-network.stories.tsx` — update.
- `web/src/hooks/routes/use-settings-*.ts` — consumed unchanged.

### Dependent Files

- `web/src/systems/settings/components/**` — consumed (task 30 output).
- `web/src/integrations/tanstack-query/**` — MSW handlers untouched; mutation contracts assert the PATCH / POST / DELETE payloads remain identical.
- `web/e2e/settings-*.spec.ts` — snapshot baselines regenerate here.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md) — primitive imports.
- [ADR-002: Greenfield migration — delete without backwards-compat](adrs/adr-002.md) — delete legacy imports with the rewrite.
- [ADR-004: Phased rollout](adrs/adr-004.md) — Phase 6 batch 2, closes the migration.
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md) — per-route snapshot baselines.

## Deliverables

- Five rewritten route files consuming only `@agh/ui` primitives and the task-30 Settings shell.
- Updated `.stories.tsx` files with idle / dirty / loading / empty variants exercised for each page.
- Playwright snapshot baselines for `/settings/mcp-servers`, `/settings/hooks-extensions`, `/settings/observability`, `/settings/environments`, `/settings/network` in idle + dirty states, plus empty baselines for the three list-oriented pages **(REQUIRED)**.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Integration tests asserting mutation payload contracts per sub-route **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `/settings/mcp-servers` — submitting the add-server `Dialog` with `{ name: "ripgrep", command: "rg", transport: "stdio" }` calls POST `/api/settings/mcp-servers` with that exact body.
  - [ ] `/settings/mcp-servers` — clicking the per-row Delete action opens a confirm `Dialog`; confirming calls DELETE `/api/settings/mcp-servers/<id>`; cancelling issues no request.
  - [ ] `/settings/mcp-servers` — when the server list is empty, the route renders `@agh/ui` `Empty` with the "No MCP servers configured" title.
  - [ ] `/settings/hooks-extensions` — toggling a hook `Switch` calls PATCH `/api/settings/hooks/<id>` with `{ enabled: true }` and marks the draft dirty.
  - [ ] `/settings/hooks-extensions` — the add-hook `Dialog` `Combobox` filters kinds (`pre-session`, `post-session`, `on-error`) and submits `{ kind: "pre-session", script: "./hooks/warmup.sh" }`.
  - [ ] `/settings/observability` — changing the log-level `NativeSelect` from `"info"` to `"debug"` calls PATCH `/api/settings` with `{ observability: { logLevel: "debug" } }` and surfaces the restart-banner in warning tone.
  - [ ] `/settings/observability` — the metric grid renders three `@agh/ui` `Metric` cards with mono eyebrows and numeric values from the hook's snapshot.
  - [ ] `/settings/environments` — toggling an environment's default `Switch` calls PATCH `/api/settings/environments/<id>` with `{ default: true }` and opens the restart-banner.
  - [ ] `/settings/environments` — opening the edit `Dialog` loads the environment's variables into editable `Field`s and submits the delta on save.
  - [ ] `/settings/network` — changing the bind-port `Input` from `2123` to `2200` marks the draft dirty and submits `{ network: { port: 2200 } }` on save.
  - [ ] `/settings/network` — toggling the TLS `Switch` on requires cert / key fields to become `Field`-required; submitting without them surfaces validation errors in the `SettingsSaveBar`.
- Integration tests:
  - [ ] Storybook `play()` on each sub-route story submits a valid change and asserts the MSW-intercepted PATCH / POST / DELETE payload matches the frozen contract.
  - [ ] Playwright snapshot baselines for the five sub-routes in idle + dirty states match within 0.1% threshold; `/settings/mcp-servers`, `/settings/hooks-extensions`, `/settings/environments` additionally match the empty-state baseline.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing and `make verify` green.
- Test coverage >=80% across the rewritten routes.
- `grep -r "from \"@/components/ui/" web/src/routes/_app/settings` returns zero hits (combined with task 31).
- `grep -r "from \"@/components/design-system" web/src/routes/_app/settings` returns zero hits (combined with task 31).
- Mutation PATCH / POST / DELETE payloads verified byte-equivalent to the pre-rewrite contract via MSW fixtures.
- Playwright baselines committed for every listed state per sub-route.
- Phase 6 end gate reached: every `/settings/*` sub-route renders under the redesigned shell with no legacy primitive imports remaining.
