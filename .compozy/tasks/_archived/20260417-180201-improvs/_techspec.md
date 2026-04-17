# TechSpec: Per-Package Improvements Pass — `internal/*`

## Executive Summary

Run a coordinated, evidence-driven improvements pass against every package under `internal/*`. Each task targets exactly one package and applies the same five analysis skills (`refactoring-analysis`, `extreme-software-optimization`, `ubs`, `deadlock-finder-and-fixer`, `security-review`). Findings that the agent decides to act on are landed as code edits in that package; everything else is recorded with a reasoned decision in a per-package report. The pass is intentionally uniform: tasks differ only in the package name they point at, which makes them safe to execute in any order, in parallel, and to re-run after the codebase moves.

This is greenfield alpha (see `CLAUDE.md` "Zero Legacy Tolerance"): no migration scaffolding, no backwards-compatibility shims, no defensive code for impossible states. Findings either get fixed cleanly or get recorded and skipped.

## Goals

- Apply systematic refactoring, performance, correctness, concurrency, and security review to every `internal/*` package.
- Land actionable fixes immediately, with tests, behind a clean `make verify`.
- Produce a durable, machine-readable trail of findings (`reports/<package>.md`) so deferred items are not lost.
- Keep the runtime's package boundaries intact — no cross-package edits, no public API breaks unless explicitly justified.

## Non-Goals

- Architectural redesign or package restructuring.
- New features. The pass strictly improves what already exists.
- Performance speculation. Optimizations need evidence (benchmark, profile, complexity argument), not vibes.
- Cross-package refactoring. If a finding lives across packages, record it and let it inform a separate effort.

## Anti-Evasion Hard Rules

These rules exist because the first executed task substituted a missing CLI for the actual skill, marked an optimization "deferred for lack of evidence" without producing the evidence, and gave a non-falsifiable "no security findings" verdict. Any of the patterns below **FAILS the task gate regardless of `make verify` result**.

1. **NO skill substitution.** If a skill cannot be invoked, mark it `not-run` in the Skill Invocation Log with the literal error. **NEVER** substitute "manual `<skill>`-style review." A manual pass is not the skill.
2. **`ubs` is a SKILL, not a CLI.** Invoke it via the Skill tool. Do NOT run `command -v ubs`. If the Skill tool refuses to invoke it, mark `ubs` as `not-run` with the exact refusal message.
3. **NO "no findings" without inventory.** Every "no findings" verdict requires the corresponding inventory (attacker-input surfaces for security, goroutine/channel/mutex tables for concurrency, hot-path candidates + benchmarks for perf, cyclomatic top-N for refactoring). The inventory must cite `file:line`.
4. **NO deferring for missing evidence the agent controls.** If a skill needs evidence (benchmark numbers, threat model, goroutine map), the agent produces it. `"Deferred — no benchmark exists"` is **forbidden** — write the benchmark, or mark the skill `not-run`.
5. **NO vague summaries.** `"Reviewed workspace path resolution"` without `file:line` citations counts as not-run for that surface.
6. **Every security verdict ships a Threat Model.** Trust boundaries, attacker capabilities, in-scope assets, out-of-scope items. No threat model = no security review done.
7. **Every security finding (HIGH or MEDIUM) traces input source → sink with `file:line`.** "Theoretical" findings are LOW and not reported.

## Methodology

Each task follows the same five-skill pipeline against one package.

| Skill | Looks for | Evidence bar before fixing |
| --- | --- | --- |
| `refactoring-analysis` | Code smells, duplication, dead code, large units, tight coupling, primitive obsession | Smell appears at least twice OR the unit exceeds repository norms; refactor preserves behavior |
| `extreme-software-optimization` | Hot paths, allocations, redundant work, algorithmic improvements | A benchmark in `*_bench_test.go` with before/after numbers OR a mechanical complexity proof. Never speculative micro-opts. |
| `ubs` | Correctness bugs, error-handling gaps, boundary issues, security smells, edge cases | Reproducible failure path or mechanical proof of incorrectness |
| `deadlock-finder-and-fixer` | Goroutine leaks, mutex misuse, channel deadlocks, missing `ctx.Done()` selects, unbounded fan-out | Identifiable hang/leak path traced through the goroutine inventory |
| `security-review` | Injection, path traversal, unsafe deserialization, command exec on untrusted input, secret handling, authn/authz gaps, SSRF, resource exhaustion | HIGH-confidence: vulnerable pattern + attacker-controllable input traced through the codebase, within the declared threat model |

Activation order is deliberate: structural findings first (refactor → optimization), then behavior-level findings (correctness → concurrency → security). Later passes often invalidate earlier ones; do the cheap reshapes before chasing performance and concurrency through code that may move.

### Mandatory Per-Skill Artifacts

Each skill produces the artifacts listed below **in the report**, regardless of whether any finding is raised. Missing artifact = skill counted as `not-run` by the gate.

#### refactoring-analysis

- **Cyclomatic top-10 table** for the package (tool: `gocyclo -over 0 internal/<pkg>/ | head -10`, or `gocyclo internal/<pkg>/ | sort -rn | head -10`). If the tool is unavailable, state the literal error and mark the skill `not-run`.
- **File-size inventory**: every `.go` file (non-test) > 300 LOC with a one-line unit-smell summary.
- **Duplication scan**: search for duplicated blocks ≥ 8 lines (`rg` heuristics, `dupl`, or equivalent). List pairs with `file:line` OR state `"scanned — no duplication ≥ 8 lines"`.
- A "nothing to fix" verdict is acceptable **only** if the three artifacts above are in the report and show nothing exceeding thresholds.

#### extreme-software-optimization

- **Hot-path candidate list**: every function on a message-handling loop, IO loop, goroutine entry point, or allocation-heavy code path, with `file:line` and reasoning.
- **Benchmarks**: for every candidate judged potentially hot, write a benchmark in `internal/<pkg>/<file>_bench_test.go` and run `go test -bench=. -benchmem -count=5 ./internal/<pkg>/...`. Capture ns/op and B/op in the report.
- **Before/after table** for every fix: benchmark name, before ns/op, before B/op, after ns/op, after B/op.
- Acceptable per-candidate outcomes: `fixed-with-benchmark | not-hot-confirmed-by-benchmark | skill-blocked-with-reason`.
- **"Deferred — no benchmark" is FORBIDDEN.** Either write the benchmark, or mark the skill `not-run` with the reason the benchmark could not be written (e.g., requires external fixture unavailable in tests).

#### ubs (Ultimate Bug Scanner)

- Invoked via the **Skill tool**. `ubs` is a skill, not a binary. Do **not** run `command -v ubs` or similar.
- Report must include an "Invocation" line with the Skill call and a short output excerpt (first and last 10 lines is sufficient) OR a `not-run` marker with the exact Skill-tool refusal message.
- **Manual "UBS-style" review is forbidden as a substitute.** If the skill cannot be invoked, the pass proceeds with ubs marked `not-run`; do not pretend to cover it manually.

#### deadlock-finder-and-fixer

- **Goroutine Inventory table**: one row per `go func(...)` / `go method(...)` in the package, columns: `file:line | owner | shutdown mechanism | notes`.
- **Channel Inventory table**: one row per channel declared in the package, columns: `file:line | capacity | owner | closer | readers | notes`.
- **Mutex Inventory table**: one row per `sync.Mutex`/`sync.RWMutex` field, columns: `file:line | read/write heavy | protects | notes`.
- **Select audit**: any `select` statement without a `ctx.Done()` branch is listed explicitly (`file:line`) or the report states `"all selects have ctx.Done() or are input-bounded"`.
- All four artifacts ship regardless of whether any finding is raised.

#### security-review

- **Threat Model** (first section of the security block). Required fields:
  - Trust boundaries (who talks to this package, across what interface)
  - Attacker capabilities (what the adversary controls — inputs, env, files, network)
  - In-scope assets (what we're protecting)
  - Out-of-scope (explicitly not defending against — e.g., "agent subprocess is assumed trusted")
  - If the package has no external exposure, state that and justify it — but still list the goroutine/input surfaces inspected.
- **Attacker-input surface inventory**: every surface where attacker-controllable data enters the package, with `file:line`, source, sanitization step, sink.
- **Per-surface verdict**: for each surface, HIGH / MEDIUM / LOW / rejected-with-reason.
- HIGH findings are reported. MEDIUM findings are listed as "needs verification." LOW / rejected findings are not in the findings table but must appear in the surface inventory.
- A "no HIGH findings" verdict is acceptable **only** if every surface in the inventory was individually rejected with reasoning.

### Hard Constraints

- **Single-package scope** — only edit files inside `internal/<package>/`. Cross-package items go in the report.
- **No backwards-compat shims, no `// removed` comments, no renamed-but-kept identifiers** — delete the old thing.
- **No weakened tests** — never relax an assertion or skip a test to make it pass; fix the underlying issue.
- **No new abstractions** unless a finding explicitly demands one.
- **No `panic`/`log.Fatal`** in production paths; no `interface{}`/`any` where a concrete type fits; `errors.Is`/`errors.As` for matching.
- **Concurrency rules from `CLAUDE.md`** apply: every goroutine has explicit ownership and shutdown via `ctx`, no `time.Sleep` for orchestration, no fire-and-forget.
- **`make verify` is the gate** — `fmt → lint → test → build`, zero warnings, zero errors. A task is not done until it passes locally.

## Per-Package Workflow

1. **Frame the package.** Read every Go file in `internal/<package>/`. List public API, goroutine entry points, external callers (via `rg "<package>\."` from the root), tests present.
2. **Produce the five inventories first** (cyclomatic top-10, hot-path candidates, goroutine/channel/mutex tables, attacker-input surface list). These go in the report before any code changes.
3. **Run the five skills in order**, populating findings into a working table: `id | skill | severity | file:line | summary`.
4. **Write the benchmarks** required by the optimization pass. Run them. Capture numbers.
5. **Triage** every finding into one of:
   - **fixed** — implemented in this task
   - **deferred** — real but out of scope (cross-package, requires design, benchmark shows not-hot); add reason
   - **wontfix** — false positive or judged not worth it; add reason
6. **Implement fixes** within `internal/<package>/`. Update or add tests for any behavior change, any newly covered branch, and any concurrency or security fix.
7. **Re-run the relevant benchmarks** after fixes. Capture before/after in the report.
8. **Validate.** Run `make verify`. If it fails, fix root cause — never bypass.
9. **Write the report** at `.compozy/tasks/improvs/reports/<package>.md` using the schema in [Report Format](#report-format).
10. **Final verify pass** via `cy-final-verify` before flipping the task to `completed`.

### Report Format

Each per-package report uses this exact structure. **Missing sections = auto-fail the task gate.**

```markdown
# Improvements Report — internal/<package>

## Skill Invocation Log

| Skill                         | Status  | Evidence / Artifact Reference                          |
| ----------------------------- | ------- | ------------------------------------------------------ |
| refactoring-analysis          | run     | cyclomatic top-10 + file-size + duplication below      |
| extreme-software-optimization | run     | 3 benchmarks in bench_test.go, numbers below           |
| ubs                           | run     | Skill tool invoked; output excerpt below               |
| deadlock-finder-and-fixer     | run     | goroutine/channel/mutex/select tables below            |
| security-review               | run     | threat model + 4 surfaces analyzed below               |

Allowed status values: `run | not-run`. `not-run` rows must include the literal error / refusal in the Evidence column. Any `run` row missing its artifact section below = auto-fail.

## Inventories

### Refactoring — Cyclomatic Top-10
(output from gocyclo)

### Refactoring — Files > 300 LOC
(file | LOC | unit-smell summary)

### Refactoring — Duplication
(file:line ↔ file:line pairs, OR "none ≥ 8 lines")

### Optimization — Hot-Path Candidates
(function | file:line | reasoning | benchmark name)

### Optimization — Benchmark Results

(benchmark | before ns/op | before B/op | after ns/op | after B/op | decision)

### UBS Invocation Output
(excerpt OR "not-run — <reason>")

### Concurrency — Goroutine Inventory
(file:line | owner | shutdown | notes)

### Concurrency — Channel Inventory
(file:line | capacity | owner | closer | readers | notes)

### Concurrency — Mutex Inventory
(file:line | read/write | protects | notes)

### Concurrency — Select Audit
(selects without ctx.Done() OR "all selects ctx-aware")

### Security — Threat Model
- Trust boundaries:
- Attacker capabilities:
- In-scope assets:
- Out-of-scope:

### Security — Attacker-Input Surface Inventory
(file:line | source | sanitization | sink | verdict)

## Findings

| ID  | Skill                             | Severity | File:Line                | Summary                       | Decision  |
| --- | --------------------------------- | -------- | ------------------------ | ----------------------------- | --------- |
| 01  | refactoring-analysis              | medium   | foo.go:42                | duplicate validation block    | fixed     |
| 02  | extreme-software-optimization     | low      | bar.go:118               | allocation in hot loop (+30%) | fixed     |
| 03  | ubs                               | high     | baz.go:77                | error swallowed on rollback   | fixed     |
| 04  | deadlock-finder-and-fixer         | high     | worker.go:55             | missing ctx.Done in select    | fixed     |
| 05  | security-review                   | high     | api.go:201               | path join with user input     | fixed     |

Severity: `critical | high | medium | low`. Every non-fixed row carries a one-line rationale in "Per-Skill Notes".

## Per-Skill Notes

### refactoring-analysis
- …

### extreme-software-optimization
- …

### ubs
- …

### deadlock-finder-and-fixer
- …

### security-review
- …

## Deferred Items (carry forward)

- **<id>** — <reason, what it would take, where it should live>

## `make verify`

Captured output excerpt (first pass fail — if any — and final clean pass):

```

## Failure Modes (auto-fail the task)

A task fails the gate (regardless of `make verify`) if any of the following is true:

1. Skill Invocation Log is missing, or a skill marked `run` has no matching artifact section below.
2. Any skill marked `run` was substituted by manual review without invoking the actual skill.
3. `extreme-software-optimization` is marked `run` without benchmark numbers in the report.
4. `security-review` is marked `run` without a Threat Model **and** attacker-input surface inventory.
5. `deadlock-finder-and-fixer` is marked `run` without Goroutine / Channel / Mutex / Select inventories.
6. `refactoring-analysis` is marked `run` without cyclomatic top-10, file-size, and duplication artifacts.
7. Any "no findings" verdict lacks the corresponding inventory of what was inspected.
8. Any code change exists outside `internal/<pkg>/`.
9. Any existing test was weakened, skipped, or removed without replacing with equal-or-stronger coverage.
10. Any `panic`, `log.Fatal`, or backwards-compat shim introduced.

## Test & Coverage Expectations

- All existing tests continue to pass with `-race`.
- Every fixed correctness, concurrency, or security finding has a covering test (table-driven; `t.Parallel()` where independent).
- Every performance fix has a benchmark in `*_bench_test.go` with pre/post numbers in the report.
- Package coverage must not regress; if currently below 80%, lift it toward 80%.
- Integration tests under `//go:build integration` are touched only when a finding directly affects them.

## Out-of-Scope Packages

None. Every directory under `internal/*` gets a task, even small utilities (`logger`, `version`, `procutil`, `fileutil`, `frontmatter`, `extensiontest`, `testutil`). For low-complexity packages the pass may yield small reports — the inventory artifacts still ship.

## Execution Order

Tasks are alphabetized by package and have no inter-task dependencies. Run in any order.

## Commit Strategy

- One commit per task.
- Title format: `refactor(<package>): improvements pass`
- Body lists fixed findings and links to the per-package report.
- Never amend across tasks.

## Success Criteria (Whole Effort)

- Every package under `internal/*` has a `reports/<package>.md` containing every mandatory inventory section.
- Every report's Skill Invocation Log marks each skill `run` (or explicitly `not-run` with reason).
- Every report's `make verify` section reads `pass`.
- No backwards-compat scaffolding, no weakened tests, no cross-package leakage introduced.
- Deferred items aggregated for follow-up planning.
