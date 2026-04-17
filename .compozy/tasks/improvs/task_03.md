---
status: pending
title: "Improvements pass — internal/automation"
type: backend
complexity: medium
dependencies: []
---

# Task 03: Improvements pass — internal/automation

## Overview

Run the five-skill improvements pass on `internal/automation/`. Apply discovered fixes directly inside that package, write a per-package report with all mandatory inventory sections, and prove `make verify` still passes. Methodology, evidence bar, anti-evasion rules, report schema, and gate failure modes are defined in `_techspec.md` — do not duplicate them here, and do not soften them.

<critical>
- ALWAYS READ `_techspec.md` in this folder before starting — pay special attention to "Anti-Evasion Hard Rules", "Mandatory Per-Skill Artifacts", and "Failure Modes (auto-fail the task)"
- ACTIVATE the five analysis skills, in order: $refactoring-analysis, $extreme-software-optimization, $ubs, $deadlock-finder-and-fixer, $security-review
- ALSO ACTIVATE $golang-pro (mandatory for any Go change), $no-workarounds, $testing-anti-patterns (when tests change), $systematic-debugging (when fixing a bug), $cy-final-verify (before marking complete)
- `ubs` is a SKILL, not a CLI. Invoke via the Skill tool. NEVER `command -v ubs` and NEVER substitute "manual UBS-style review". If the Skill tool refuses, mark `ubs` as `not-run` with the literal refusal message in the Skill Invocation Log.
- `extreme-software-optimization` REQUIRES benchmarks. Write `*_bench_test.go` for every hot-path candidate, run `go test -bench=. -benchmem -count=5`, capture before/after numbers in the report. "Deferred — no benchmark" is FORBIDDEN.
- `security-review` REQUIRES an explicit Threat Model section AND an attacker-input surface inventory before any "no findings" verdict is allowed. List every surface with file:line.
- `deadlock-finder-and-fixer` REQUIRES Goroutine / Channel / Mutex / Select inventories in the report regardless of whether any finding is raised.
- `refactoring-analysis` REQUIRES cyclomatic top-10 + files-over-300-LOC + duplication scan in the report regardless of whether any finding is raised.
- SCOPE IS THIS PACKAGE ONLY — only edit files inside `internal/automation/`. Cross-package findings go in the report under "Deferred Items". Editing outside the package = auto-fail.
- NO backwards-compat shims, no defensive code for impossible cases, no migration scaffolds (greenfield alpha — see `CLAUDE.md` "Zero Legacy Tolerance")
- NEVER weaken, skip, or remove a test to make a fix pass — fix the root cause
- `make verify` MUST pass at the end — fmt → lint → test → build, zero warnings, zero errors
- ANY non-trivial finding NOT fixed must be recorded in the Findings table AND in "Per-Skill Notes" with reasoning
</critical>

<requirements>
- MUST produce all five mandatory inventory sections in `.compozy/tasks/improvs/reports/automation.md` BEFORE the Findings table:
    1. Refactoring: cyclomatic top-10, files > 300 LOC, duplication scan
    2. Optimization: hot-path candidate list and benchmark results table
    3. UBS: Skill invocation evidence (output excerpt or not-run reason)
    4. Concurrency: goroutine inventory, channel inventory, mutex inventory, select audit
    5. Security: threat model + attacker-input surface inventory
- MUST populate the Skill Invocation Log table at the top of the report with `run | not-run` per skill and a pointer to the artifact section
- MUST write benchmarks in `internal/automation/`-co-located `*_bench_test.go` files for every hot-path candidate identified
- MUST trace input source → sink with file:line for every security finding (HIGH or MEDIUM)
- MUST triage every finding into `fixed | deferred | wontfix` with a one-line reason for non-fixed items
- MUST land every "fixed" finding as code edits inside `internal/automation/`
- MUST update or add tests for any behavior change, fixed correctness or concurrency bug, and any newly covered branch
- MUST add a benchmark with pre/post numbers for every performance fix
- MUST keep public package API stable unless a finding explicitly justifies a break (record the break in the report)
- MUST NOT introduce new abstractions unless a finding demands one
- MUST capture the final `make verify` excerpt in the report
- SHOULD lift package coverage toward 80% if currently below
- SHOULD prefer fewer, deeper fixes over many shallow tweaks
</requirements>

## Subtasks

- [ ] 03.1 Read `_techspec.md` end-to-end (especially Anti-Evasion Hard Rules, Mandatory Per-Skill Artifacts, Failure Modes); confirm scope
- [ ] 03.2 Map `internal/automation/`: list every Go file, public surface, goroutine entry points, and external callers (rg from repo root)
- [ ] 03.3 Build the five mandatory inventories (cyclomatic top-10, hot-path candidates, goroutine/channel/mutex/select tables, attacker-input surfaces) and write them to the report BEFORE running fixes
- [ ] 03.4 Write benchmarks (`*_bench_test.go`) for every hot-path candidate; run `go test -bench=. -benchmem -count=5 ./internal/automation/...`; capture baseline numbers
- [ ] 03.5 Run $refactoring-analysis against the file-size + duplication + cyclomatic inventory; record findings
- [ ] 03.6 Run $extreme-software-optimization against the benchmarked candidates; record findings (only fixes with measurable improvement count as "fixed")
- [ ] 03.7 Invoke $ubs via the Skill tool; capture output excerpt OR mark not-run with literal refusal message
- [ ] 03.8 Run $deadlock-finder-and-fixer against the goroutine/channel/mutex/select inventories; record findings
- [ ] 03.9 Run $security-review after writing the threat model and the attacker-input surface inventory; per-surface verdict required
- [ ] 03.10 Triage all findings into fixed / deferred / wontfix with reasons
- [ ] 03.11 Apply fixes (Go code + tests + benchmark deltas) inside `internal/automation/`
- [ ] 03.12 Run `make verify`; fix root causes until clean; capture final excerpt
- [ ] 03.13 Re-run benchmarks; populate before/after numbers in the optimization table
- [ ] 03.14 Verify the report against `_techspec.md` "Failure Modes" — every `run` skill has its artifact section; every "no findings" carries an inventory
- [ ] 03.15 Run $cy-final-verify before flipping status

## Implementation Details

See `_techspec.md` ("Methodology", "Mandatory Per-Skill Artifacts", "Per-Package Workflow", "Report Format", "Failure Modes") for the shared methodology, evidence bar, and gate. Do not duplicate techspec content here, and do not soften it.

### Relevant Files

- `internal/automation/` — sole edit scope (all Go files, including tests and new `*_bench_test.go`)

### Report Location

- `.compozy/tasks/improvs/reports/automation.md` — must include every mandatory inventory section per techspec

## Deliverables

- All "fixed" findings applied as code changes in `internal/automation/`
- Tests added or updated for behavior changes and any fixed correctness, concurrency, or security bug
- Benchmarks added in `*_bench_test.go` for every hot-path candidate, with before/after numbers in the report
- `.compozy/tasks/improvs/reports/automation.md` populated with: Skill Invocation Log, all five inventory sections, Findings table, Per-Skill Notes, Deferred Items, `make verify` excerpt
- `make verify` passing cleanly

## Tests

- Existing tests pass with `-race`
- New tests required for any fixed correctness, concurrency, or security finding
- Benchmarks (`*_bench_test.go`) for every hot-path candidate
- Package coverage does not regress; aim toward 80% if currently below

## Success Criteria

- `make verify` passes (fmt → lint → test → build), zero warnings, zero errors
- `.compozy/tasks/improvs/reports/automation.md` exists and passes the techspec "Failure Modes" checklist:
    1. Skill Invocation Log present with status per skill
    2. Every `run` skill has its artifact section
    3. `extreme-software-optimization` shows benchmark numbers (or skill is `not-run`)
    4. `security-review` shows Threat Model + attacker-input inventory (or skill is `not-run`)
    5. `deadlock-finder-and-fixer` shows Goroutine/Channel/Mutex/Select inventories (or skill is `not-run`)
    6. `refactoring-analysis` shows cyclomatic top-10 + file-size + duplication (or skill is `not-run`)
    7. Every "no findings" verdict carries the corresponding inventory
- No edits outside `internal/automation/`
- No backwards-compat shims, no weakened tests, no new speculative abstractions
- No `ubs` substitution; no security verdict without threat model; no perf "fix" without benchmark numbers
