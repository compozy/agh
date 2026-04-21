# Harness Runtime — Regression Suite

**Feature:** Harness runtime architecture
**Date:** 2026-04-18
**Status:** Active
**QA Output Path:** `.compozy/tasks/harness`

---

## 1. Purpose and Scope

This regression suite converts the harness QA plan into an execution matrix for
`task_10`. It prioritizes real daemon/runtime seams over generic coverage and
keeps the artifact contract fixed under `.compozy/tasks/harness/qa/`.

The suite is organized around the runtime decisions that can regress silently:

- startup selection from durable session context
- ordered prompt augmentation
- synthetic prompt persistence and trust boundaries
- detached task-runtime mapping
- completion-driven synthetic reentry or observable drop
- event-summary visibility and HTTP/UDS parity
- boot/restart recovery and duplicate-protection

---

## 2. P0 / P1 Execution Matrix

| Test ID | Priority | Seam | Traceability | Primary Suites |
| --- | --- | --- | --- | --- |
| `TC-INT-001` | P0 | Context resolution + startup section selection | Workstream 1/2, `task_01.md`, `task_02.md` | Smoke, full |
| `TC-INT-002` | P0 | Ordered augmentation + stored-input invariant | Workstream 3, `task_03.md` | Smoke, targeted, full |
| `TC-INT-003` | P0 | Synthetic prompt persistence + FIFO queueing | Workstream 4, `task_04.md` | Targeted, full |
| `TC-INT-004` | P0 | Transcript / hook / extension synthetic trust | Workstream 4, `task_05.md` | Targeted, full |
| `TC-INT-005` | P0 | Detached task-runtime mapping + idempotency | Workstream 5, `task_06.md` | Targeted, full |
| `TC-INT-006` | P0 | Completion-to-reentry bridge + drop rules | Workstream 4/5, `task_07.md` | Smoke, targeted, full |
| `TC-INT-007` | P0 | Harness observability + HTTP/UDS parity | Workstream 6, `task_08.md` | Smoke, targeted, full |
| `TC-REG-001` | P1 | Restart / recovery / duplicate-protection | Workstream 5/6, `task_06.md` to `task_08.md` | Targeted, full |

---

## 3. Smoke Suite

**Target duration:** 20-30 minutes  
**Frequency:** first execution on `task_10`, per risky harness change, before
deeper targeted/full regression  
**Gate rule:** if any smoke item fails, stop immediately and fix before deeper
execution

| Order | Item | Type | Priority | Why it is in smoke |
| --- | --- | --- | --- | --- |
| 1 | Baseline repo gate (`make verify`) | Command gate | P0 | Confirms the branch is at least build/lint/test clean before runtime QA |
| 2 | `TC-INT-001` | Runtime case | P0 | Startup policy and network overlay are the first visible harness contract |
| 3 | `TC-INT-002` | Runtime case | P0 | Augmentation can silently corrupt prompt semantics if broken |
| 4 | `TC-INT-006` | Runtime case | P0 | Detached completion to reentry is the highest-risk cross-workstream seam |
| 5 | `TC-INT-007` | Runtime case | P0 | Observability/parity proves the operator can actually see harness decisions |

### Smoke Stop Conditions

- `make verify` fails
- startup selection is wrong or duplicated
- stored input is overwritten by augmentation
- completion emits duplicate or missing synthetic wake behavior
- HTTP and UDS disagree on harness-visible event summaries

---

## 4. Targeted Regression

**Target duration:** 30-60 minutes per changed area  
**Frequency:** after any harness change or bug fix  
**Rule:** run smoke first, then run only the cases mapped to the changed seam(s)

### 4.1 Context / Startup Changes

Use when changes touch `internal/daemon/harness_context.go`,
`internal/daemon/section_selector.go`, `internal/daemon/prompt_sections.go`,
`internal/daemon/composed_assembler.go`, or equivalent startup wiring.

- Run `TC-INT-001`
- Run `TC-INT-007`
- Run `TC-REG-001` if the change affects resume, boot, or summary replay

### 4.2 Augmentation Changes

Use when changes touch prompt augmentation composition, budget/failure policy,
or the manager prompt path.

- Run `TC-INT-002`
- Run `TC-INT-007`

### 4.3 Synthetic Input / Transcript Changes

Use when changes touch synthetic input persistence, transcript assembly, hook
classification, or extension-host replay.

- Run `TC-INT-003`
- Run `TC-INT-004`
- Run `TC-INT-007`

### 4.4 Detached Runtime / Reentry Changes

Use when changes touch `internal/daemon/task_runtime.go`,
`internal/daemon/harness_detached_work.go`,
`internal/daemon/harness_reentry_bridge.go`, task-run metadata, or recovery
logic.

- Run `TC-INT-005`
- Run `TC-INT-006`
- Run `TC-INT-007`
- Run `TC-REG-001`

### 4.5 Observe / Transport Changes

Use when changes touch `internal/observe`, `internal/store/globaldb`
event-summary logic, HTTP/UDS stream/read handlers, or parity surfaces.

- Run `TC-INT-007`
- Re-run the smoke suite
- Run `TC-REG-001` if boot/restart behavior or summary replay changed

---

## 5. Full Regression

**Target duration:** 2-4 hours  
**Frequency:** before release, after major refactors, or when smoke/targeted
results suggest cross-seam risk

### Full Execution Order

1. Run the Smoke Suite.
2. Run remaining P0 cases in this order:
   - `TC-INT-003`
   - `TC-INT-004`
   - `TC-INT-005`
3. Run P1 recovery coverage:
   - `TC-REG-001`
4. Run the harness-wide integration bundle expected by `task_10`:
   - `go test -tags integration ./internal/daemon ./internal/api/httpapi ./internal/api/udsapi -count=1`
5. Re-run the broad integration gate when fixes or broad harness changes landed:
   - `make test-integration`
6. Finish with a fresh final gate:
   - `make verify`

### Full-Suite Expectations

- Every harness workstream has direct runtime evidence, not just package-local
  unit confidence.
- Recovery and duplicate-protection are proven after a restart boundary.
- Final verification is always fresh after the last fix.

---

## 6. Evidence and Output Rules

All execution outputs stay under `.compozy/tasks/harness/qa/`.

### Required Evidence Targets

- `verification-report.md`
  - smoke results
  - targeted/full results
  - event-summary sequence tables
  - HTTP/UDS parity comparison
  - restart/recovery comparison
  - final verdict
- `issues/BUG-*.md`
  - one file per discovered defect
  - must reference the originating `TC-*` id
- `screenshots/`
  - only when browser/UI evidence is genuinely relevant

### Minimum Per-Case Evidence

- command or runtime lane used
- pass/fail outcome
- IDs of the affected session/task/task_run when relevant
- the specific observable proof:
  - transcript excerpt
  - stored event payload
  - ordered `harness.*` summaries
  - HTTP/UDS response excerpt
  - recovery/dedupe state before and after restart

---

## 7. Pass / Fail Criteria

| Verdict | Criteria |
| --- | --- |
| PASS | All P0 cases pass, 90%+ P1 cases pass, no Critical/High harness bug remains open, and final `make verify` passes |
| FAIL | Any P0 case fails, a synthetic trust violation is found, a duplicate wakeup occurs, transport parity diverges, or final verification fails |
| CONDITIONAL | Only P1 failures remain, each has a documented workaround and fix plan, and no P0/runtime-integrity risk remains |

Additional automatic fail conditions:

- any harness-visible behavior is only provable by reading code instead of a
  runtime artifact
- any bug is “accepted” without a `BUG-*` artifact or root-cause fix
- the final verification report lacks command evidence for the completion claim

---

## 8. Competitor-Inspired Focus Areas

- Hermes checkpoint/resumption drives `TC-REG-001` so restart/recovery is not
  treated as optional.
- OpenFang’s inspectable event/API surfaces inform `TC-INT-007`; harness
  runtime decisions must stay externally inspectable.
- Claude Code’s background-task completion patterns inform `TC-INT-006`; the
  bridge from detached completion back to the foreground session must remain
  explicit and auditable.

---

## 9. Task 10 Handoff

`task_10` should start from this order unless the user explicitly overrides it:

1. `make verify`
2. `TC-INT-001`
3. `TC-INT-002`
4. `TC-INT-006`
5. `TC-INT-007`
6. remaining P0 cases
7. `TC-REG-001`
8. final integration / verification gates

No output paths should change during execution.
