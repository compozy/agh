# Settings UI Regression Suite

**Feature:** Settings UI
**Suite owner:** `task_16.md`
**Source plan:** `qa/test-plans/settings-ui-test-plan.md`
**Case inventory:** `qa/test-cases/`
**Execution output:** `qa/verification-report.md`, `qa/screenshots/`, `qa/issues/`

## Purpose

This regression document defines the reusable execution lanes for the settings feature. It preserves the P0/P1 priorities and the stop/go rules that `task_16` must follow.

## Execution Order

1. Smoke suite
2. Affected targeted suite
3. Remaining P0 cases not already exercised
4. Remaining P1 cases
5. UI visual cases
6. Exploratory follow-up around any failing area

If the smoke suite fails, stop and fix the blocking issue before progressing.

## Case Priority Catalog

### P0

- `TC-FUNC-001` Settings shell navigation and section entrypoints
- `TC-FUNC-002` General restart-required save and daemon restart flow
- `TC-FUNC-005` Skills applied-now vs restart-required behavior
- `TC-FUNC-008` Providers CRUD and builtin fallback behavior
- `TC-FUNC-010` MCP servers global-scope precedence and target selection
- `TC-INT-011` MCP servers workspace scope and cache isolation
- `TC-FUNC-012` Hooks & Extensions hybrid behavior
- `TC-INT-013` Non-loopback HTTP mutation restriction messaging

### P1

- `TC-FUNC-003` Memory restart-aware config and consolidate action
- `TC-FUNC-004` Observability runtime diagnostics and log-tail capability
- `TC-FUNC-006` Automation summary page and restart-aware save
- `TC-FUNC-007` Network summary page and operational deep-link behavior
- `TC-FUNC-009` Environments CRUD and usage-count handling
- `TC-UI-014` Summary route visual validation against Paper exports
- `TC-UI-015` Collection and hybrid route visual validation against Paper exports

## Smoke Suite

**Goal:** Prove the settings feature is alive, navigable, and safe to test further.

**Target duration:** 20-30 minutes

| Order | Case ID | Why it is in smoke |
|------|---------|--------------------|
| 1 | `TC-FUNC-001` | Proves the shell, nav order, and route entrypoints exist |
| 2 | `TC-FUNC-002` | Proves the highest-risk restart-required flow and restart polling |
| 3 | `TC-FUNC-005` | Proves the applied-now vs restart-required split on a summary page |
| 4 | `TC-FUNC-008` | Proves collection CRUD and builtin fallback on providers |
| 5 | `TC-FUNC-010` | Proves MCP precedence, target selection, and restart-required collection saves |
| 6 | `TC-INT-011` | Proves workspace-scoped MCP behavior and cache/scope separation |
| 7 | `TC-FUNC-012` | Proves Hooks & Extensions immediate-action vs restart-aware behavior |

**Smoke stop conditions**

- Any smoke case fails.
- Restart status never reaches a terminal state when the environment should support restart.
- A mutation silently succeeds without visible applied-now or restart-required messaging.
- A collection mutation changes the wrong scope or source.

## Targeted Suites

### Targeted Suite A: Summary and Restart Semantics

**When to run:** Changes in route shells, save-bar logic, restart banner logic, or summary-page hooks

**Cases**

- `TC-FUNC-001`
- `TC-FUNC-002`
- `TC-FUNC-003`
- `TC-FUNC-004`
- `TC-FUNC-005`
- `TC-FUNC-006`
- `TC-FUNC-007`
- `TC-INT-013`
- `TC-UI-014`

### Targeted Suite B: Collection CRUD and Precedence

**When to run:** Changes in providers, environments, MCP servers, collection dialogs, or write-target semantics

**Cases**

- `TC-FUNC-008`
- `TC-FUNC-009`
- `TC-FUNC-010`
- `TC-INT-011`
- `TC-UI-015`

### Targeted Suite C: Hooks, Extensions, and HTTP Restriction Policy

**When to run:** Changes in hooks, extensions, transport guards, or mutation-availability messaging

**Cases**

- `TC-FUNC-012`
- `TC-INT-013`
- `TC-UI-015`

## Full Suite

**Goal:** Complete settings feature release gate

**Target duration:** 2-4 hours, including screenshots and bug documentation

**Cases**

- `TC-FUNC-001`
- `TC-FUNC-002`
- `TC-FUNC-003`
- `TC-FUNC-004`
- `TC-FUNC-005`
- `TC-FUNC-006`
- `TC-FUNC-007`
- `TC-FUNC-008`
- `TC-FUNC-009`
- `TC-FUNC-010`
- `TC-INT-011`
- `TC-FUNC-012`
- `TC-INT-013`
- `TC-UI-014`
- `TC-UI-015`

## Post-Fix Sanity Suite

**When to run:** After a blocking fix discovered during `task_16`

**Minimum rerun**

- The failing case
- The matching targeted suite
- Any directly coupled P0 case

**Suggested default**

- `TC-FUNC-001`
- `TC-FUNC-002`
- `TC-FUNC-010`
- `TC-INT-011`
- `TC-FUNC-012`

## Pass / Fail Criteria

### PASS

- All P0 cases pass.
- At least 90% of P1 cases pass.
- No `Critical` or `High` settings bugs remain open without a documented workaround and fix plan.
- Evidence artifacts exist for restart, collection, workspace, and hooks/extensions flows.

### FAIL

- Any P0 case fails.
- A restart flow loses status continuity or reports misleading terminal state.
- A collection CRUD action writes to the wrong scope or hides fallback behavior.
- Non-loopback HTTP mutation restrictions are missing, bypassed, or misleading.
- Immediate extension actions trigger restart-required UX or vice versa.

### CONDITIONAL PASS

- One or more P1 cases fail, but:
  - no P0 case fails,
  - the failure has a documented workaround,
  - a bug report exists under `qa/issues/`,
  - and the fix plan is recorded in `qa/verification-report.md`.

## Evidence Requirements

- `qa/verification-report.md` must list the executed suite name, environment, case IDs, pass/fail result, and rerun status.
- Screenshots are required for:
  - `TC-FUNC-002`
  - `TC-FUNC-010`
  - `TC-INT-011`
  - `TC-FUNC-012`
  - `TC-INT-013`
  - `TC-UI-014`
  - `TC-UI-015`
- Bugs discovered during any suite must be documented as `BUG-*.md` and linked from the verification report.

## Handoff to `task_16`

- Use the smoke suite to choose the first committed browser E2E coverage candidates.
- Keep case IDs unchanged in Playwright spec comments, verification reporting, and bug references.
- Do not downgrade a P0 case to P1 during execution. If the environment blocks a P0 case, mark it `Blocked` with evidence and resolve the blocker before sign-off.
