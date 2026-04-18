# Settings UI QA Verification Report

**Feature:** Settings UI
**Execution date:** 2026-04-17
**Execution owner:** `task_16.md`
**QA output path:** `.compozy/tasks/settings-ui/qa/`
**Execution workflow:** `/qa-execution` with `qa-output-path=.compozy/tasks/settings-ui`
**Source matrix:** `qa/test-plans/settings-ui-test-plan.md`, `qa/test-plans/settings-ui-regression.md`, `qa/test-cases/`

## Environment

- Local repository checkout on macOS using the repo-managed daemon-served browser harness.
- Browser lane: Playwright Chromium through `web/e2e/fixtures/test.ts`.
- Bind variants exercised:
  - loopback `127.0.0.1` for positive mutation and restart flows
  - non-loopback `0.0.0.0` for ADR-004 restriction behavior
- Workspace fixture: `ws-polybot`
- Seed path: `web/e2e/fixtures/runtime-seed.ts`
- Verification gates:
  - `make test-e2e-web`
  - `make verify`
  - `go test -count=1 -tags integration ./internal/api/udsapi -run TestUDSTransportSettingsMutationsRemainPrivilegedWhenHTTPIsNonLoopback`

## Execution Notes

- The task text references `scripts/discover-project-contract.py`, but that file does not exist in this checkout. Execution used the repo's actual verification contract (`Makefile` plus existing Go/Web test lanes) and treated the missing script as a task-document mismatch, not a product defect.
- Settings browser coverage was committed into the normal daemon-served Playwright lane under `web/e2e/`; no parallel-only browser harness was introduced.
- Three blocking defects were discovered during execution, fixed at the source, and covered with matching regression tests before the final reruns.

## Executed Suites

### Smoke Suite

Daemon-served Playwright coverage landed in:

- `web/e2e/settings.spec.ts`
- `web/e2e/settings-transport.spec.ts`

Executed P0 smoke cases:

- `TC-FUNC-001` via settings shell navigation coverage
- `TC-FUNC-002` via restart-required save, restart polling, and refresh continuity
- `TC-FUNC-005` via skills applied-now vs restart-required split
- `TC-FUNC-008` via providers CRUD and builtin fallback
- `TC-FUNC-010` via MCP target and precedence coverage
- `TC-INT-011` via workspace-scoped MCP flow
- `TC-FUNC-012` via hooks/extensions hybrid behavior
- `TC-INT-013` via non-loopback HTTP restriction messaging

Result: `PASS`

### Targeted Post-Fix Coverage

The following focused regression commands were used during the bug-fix loop:

- `go test ./internal/config ./internal/settings ./internal/daemon -count=1`
- `bun x vitest run src/systems/settings/hooks/use-settings-restart.test.tsx src/systems/settings/hooks/use-settings-mutations.test.tsx src/hooks/routes/use-settings-page.test.tsx src/hooks/routes/use-settings-general-page.test.tsx src/hooks/routes/use-settings-memory-page.test.tsx src/hooks/routes/use-settings-observability-page.test.tsx src/hooks/routes/use-settings-automation-page.test.tsx src/hooks/routes/use-settings-network-page.test.tsx src/hooks/routes/use-settings-skills-page.test.tsx src/hooks/routes/use-settings-providers-page.test.tsx src/hooks/routes/use-settings-environments-page.test.tsx src/hooks/routes/use-settings-mcp-servers-page.test.tsx src/hooks/routes/use-settings-hooks-extensions-page.test.tsx`
- `go test -count=1 -tags integration ./internal/api/udsapi -run TestUDSTransportSettingsMutationsRemainPrivilegedWhenHTTPIsNonLoopback`

Result: `PASS`

### Final Repository Gates

- `make test-e2e-web` -> `PASS`
- `make verify` -> `PASS`

These reruns were executed after the last production fix and after the new settings Playwright coverage was part of the browser lane.

## Case Matrix

| Case ID | Title | Result | Execution path | Evidence |
|------|-------|--------|----------------|----------|
| `TC-FUNC-001` | Settings shell navigation and section entrypoints | PASS | `web/e2e/settings.spec.ts` | `qa/screenshots/TC-FUNC-001-settings-shell-navigation.png` |
| `TC-FUNC-002` | General restart-required save and daemon restart flow | PASS | `web/e2e/settings.spec.ts` plus restart store regression test | `qa/screenshots/TC-FUNC-002-general-restart-polling.png`, `qa/screenshots/TC-INT-016-general-restart-ready.png`, `qa/issues/BUG-001-restart-refresh-continuity.md` |
| `TC-FUNC-003` | Memory restart-aware config and consolidate action | PASS | route-hook regression lane in `make verify` | covered by focused Vitest rerun plus final `make verify` |
| `TC-FUNC-004` | Observability runtime diagnostics and log-tail capability | PASS | route-hook regression lane in `make verify` | covered by focused Vitest rerun plus final `make verify` |
| `TC-FUNC-005` | Skills applied-now vs restart-required behavior | PASS | `web/e2e/settings.spec.ts` plus persistence regression tests | `qa/screenshots/TC-FUNC-005-skills-applied-now-vs-restart.png`, `qa/issues/BUG-002-skills-overlay-persistence.md` |
| `TC-FUNC-006` | Automation summary page and restart-aware save | PASS | route-hook regression lane in `make verify` | covered by focused Vitest rerun plus final `make verify` |
| `TC-FUNC-007` | Network summary page and operational deep-link behavior | PASS | route-hook regression lane in `make verify` | covered by focused Vitest rerun plus final `make verify` |
| `TC-FUNC-008` | Providers CRUD and builtin fallback behavior | PASS | `web/e2e/settings.spec.ts` | `qa/screenshots/TC-FUNC-008-providers-crud-and-builtin-fallback.png` |
| `TC-FUNC-009` | Environments CRUD and usage-count handling | PASS | route-hook regression lane in `make verify` | covered by focused Vitest rerun plus final `make verify` |
| `TC-FUNC-010` | MCP servers global-scope precedence and target selection | PASS | `web/e2e/settings.spec.ts` | `qa/screenshots/TC-FUNC-010-mcp-global-precedence.png` |
| `TC-INT-011` | MCP servers workspace scope and cache isolation | PASS | `web/e2e/settings.spec.ts` | `qa/screenshots/TC-INT-011-mcp-workspace-scope.png` |
| `TC-FUNC-012` | Hooks & Extensions hybrid behavior | PASS | `web/e2e/settings.spec.ts` plus daemon transport parity regression tests | `qa/screenshots/TC-FUNC-012-hooks-extensions-hybrid.png`, `qa/issues/BUG-003-settings-transport-parity.md` |
| `TC-INT-013` | Non-loopback HTTP mutation restriction messaging | PASS | `web/e2e/settings-transport.spec.ts` plus UDS integration rerun | `qa/screenshots/TC-INT-013-non-loopback-http-restrictions.png` |
| `TC-UI-014` | Summary route visual validation against Paper exports | PASS | daemon-served shell and navigation capture | `qa/screenshots/TC-UI-014-summary-shell-desktop.png` |
| `TC-UI-015` | Collection and hybrid route visual validation against Paper exports | PASS | daemon-served providers and hooks/extensions captures | `qa/screenshots/TC-UI-015-collection-providers-desktop.png`, `qa/screenshots/TC-UI-015-hybrid-hooks-extensions-desktop.png` |

## Defects Found and Fixed

### `BUG-001` Restart polling lost continuity after full page refresh

- Origin: `TC-FUNC-002`
- Severity: `High`
- Status: `Fixed`
- Report: `qa/issues/BUG-001-restart-refresh-continuity.md`
- Root cause: the restart banner and polling state lived only in in-memory Zustand state, so a full document refresh during daemon replacement cleared the active operation id and hid the in-progress restart state.
- Fix:
  - persisted restart store state in `sessionStorage`
  - added rehydration-aware regression coverage for the restart hook
  - updated browser coverage to prove refresh continuity against the real daemon restart flow

### `BUG-002` Nested skills overlay writes produced invalid TOML

- Origin: `TC-FUNC-005`
- Severity: `Critical`
- Status: `Fixed`
- Report: `qa/issues/BUG-002-skills-overlay-persistence.md`
- Root cause: nested settings overlays like `[skills.marketplace]` were rendered through a broken fragment path that produced malformed TOML and failed HTTP saves.
- Fix:
  - normalized nested tree values before `tomltree.TreeFromMap`
  - corrected overlay line-end offsets
  - fixed nested table rendering in overlay fragments
  - added regression coverage in `internal/config/persistence_test.go`

### `BUG-003` Hooks/extensions transport parity was reported as unknown on loopback

- Origin: `TC-FUNC-012`, `TC-INT-013`
- Severity: `High`
- Status: `Fixed`
- Report: `qa/issues/BUG-003-settings-transport-parity.md`
- Root cause: daemon runtime transport parity returned a zero-value status instead of deriving settings and extension mutation availability from the actual bind host.
- Fix:
  - compute parity explicitly in `internal/daemon/settings.go`
  - add daemon regression tests covering loopback, localhost, wildcard, and non-loopback binds
  - keep non-loopback browser coverage in the standard Playwright lane

## Fresh Evidence Inventory

Screenshots captured under `qa/screenshots/`:

- `TC-FUNC-001-settings-shell-navigation.png`
- `TC-FUNC-002-general-restart-polling.png`
- `TC-FUNC-005-skills-applied-now-vs-restart.png`
- `TC-FUNC-008-providers-crud-and-builtin-fallback.png`
- `TC-FUNC-010-mcp-global-precedence.png`
- `TC-FUNC-012-hooks-extensions-hybrid.png`
- `TC-INT-011-mcp-workspace-scope.png`
- `TC-INT-013-non-loopback-http-restrictions.png`
- `TC-INT-016-general-restart-ready.png`
- `TC-UI-014-summary-shell-desktop.png`
- `TC-UI-015-collection-providers-desktop.png`
- `TC-UI-015-hybrid-hooks-extensions-desktop.png`

Bug records captured under `qa/issues/`:

- `BUG-001-restart-refresh-continuity.md`
- `BUG-002-skills-overlay-persistence.md`
- `BUG-003-settings-transport-parity.md`

## Exit Assessment

- All P0 cases passed.
- All P1 cases have automated verification coverage in the focused Vitest rerun plus the final `make verify` lane; no open P1 failure remains.
- No open `Critical` or `High` settings defects remain after the final fix set.
- The settings browser coverage now lives in the normal repo lane and passed in `make test-e2e-web`.
- Final status: `PASS`
