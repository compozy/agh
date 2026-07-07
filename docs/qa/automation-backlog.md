# Automation Backlog

Automation intent lives here only (see `.agents/skills/qa-report/references/automation-backlog.md` for entry rules). `AB-NNN` ids are monotonic.

Seed material: `_seeds/qa-e2e-playbook.md` §10 (first-automation backlog) and `_seeds/final-qa/` module plans — promote entries from there as journeys stabilize.

<!-- ## AB-NNN: <title>
- Source: <J-NN / state.csv ids / BUG-NNNN>
- Why automate: <regression-prone | high-value stable journey | fix lacked a test>
- Suggested layer: <E2E browser | API/integration | unit>
- Spec sketch: <entry, key assertions incl. true end state>
- Status: proposed -->

## AB-001: Loop web E2E seed harness (real-daemon Playwright)
- Source: J-01, J-03, J-04, J-06, J-08, J-09 / LP-001..LP-005, LP-008..LP-016, LP-021..LP-024, LP-029..LP-035 / `_tests.md` E2E-web-2..9, 12..17
- Why automate: high-value stable journeys blocked from real-daemon browser E2E — `web/e2e/fixtures/*` has NO loop seed flow that drives the rich Loop SSE states the run page binds. Covered today at daemon/runtime tests, vitest/component layer, and `agh-ui-screenshot` visual parity (see the loops shared workflow memory open risk).
- Suggested layer: E2E browser (`make test-e2e-web`) + a daemon-side loop seed fixture emitting the enumerated SSE kinds.
- Spec sketch: seed a running/needs-approval/paused/watching loop_run with generation + gate + meter frames; drive catalog→run-form→run-detail; assert contract header, meters (cost display-only), timeline, approval routing, pause→paused-at-boundary, and the truthful terminal banner. True end state: the browser view matches the seeded run and reloads without an optimistic-UI lie.
- Status: proposed

## AB-002: Converse-and-decide run seed (no installed template)
- Source: J-10 / LP-036, LP-037 / `_tests.md` E2E-web-6, Integration-21, Unit-26
- Why automate: high-value differentiator with NO runtime-installed template (docs-only, PRD F10) — the run-detail channel panel + `channel_result` harvest cannot be exercised end-to-end without a hand-built seed loop. Blocks CH-010 from running.
- Suggested layer: E2E browser (channel panel render) + integration (harvest happy-path + windowed stall) over a fake conversation store.
- Spec sketch: build a seed loop with an `agh__network_send` node declaring `harvest: {kind: channel_result, window, responder?}`; drive a conversation to a designated result → assert the harvested payload highlights in the timeline and drives the fan-out to `done`; a no-result window → assert terminal `stalled` (never a fabricated decision). True end state: the harvested decision is visible and executed, or `stalled` with escalation.
- Status: proposed

## AB-003: Agent-operability CLI↔HTTP↔UDS↔native-tool parity harness
- Source: J-07 / LP-025..LP-028 / `_tests.md` E2E-runtime-3, E2E-runtime-5, Integration-27, Integration-28
- Why automate: regression-prone cross-surface contract — every `agh__loop_*` verb must match CLI/HTTP/UDS byte-for-byte on the same inputs, the approve capability gate (no self-approval) must hold, and `Unavailable(ReasonDependencyMissing)` must be deterministic. Manual re-verification of the full verb matrix each cycle is error-prone.
- Suggested layer: API/integration (Go harness) driving the full verb set across all four surfaces + a native-tool-vs-HTTP diff assertion.
- Spec sketch: for run/dry-run/configure/pause/resume/stop/approve/list/inspect/status/runs/edit/delete — assert identical terminal outcomes and structured payloads across surfaces; assert an agent cannot approve its own gate; assert hash-form-only token redaction. True end state: structured agent operation equals operator operation (PRD 'Surface coverage').
- Status: proposed

## AB-004: All-11-status observation seeds
- Source: J-03, J-07, J-08 / LP-012, LP-026, LP-030, LP-038 / `_tests.md` E2E-runtime-10, E2E-web-9, Web-unit-5/8
- Why automate: the truthful-outcome guarantee (no coercion) needs seeds that actually produce each of the 11 states — incl. `no-op`, `blocked` (ADR-022), `queued` (ADR-021), and `paused` — which today's daemon rarely emits richly. Pins the "never render a terminal coercion" invariant.
- Suggested layer: E2E runtime (produce each state) + web component (map each state to its distinct pill, reduced-motion-gated pulse, no coercion).
- Spec sketch: drive/seed a run into each of the 11 states; assert a distinct truthful pill per state and that `exhausted`/`stalled`/`needs-approval`/`no-op`/`blocked` are never rendered as `done`/`failed`. The terminal `blocked` (external dependency impossible, ADR-022) has a `state.csv` owner (LP-038, J-08) but needs a *behavioral* seed that walks a run into it — e.g. a J-08 watch-source missing a credential, or a refused `run-loop` cycle — not only a rendered pill. True end state: all 11 states render distinctly and truthfully across the web + structured surfaces.
- Status: proposed
