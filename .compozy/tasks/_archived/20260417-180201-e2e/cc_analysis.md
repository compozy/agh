---
title: E2E Mock Strategy — Post-Implementation Analysis
status: analysis
created_at: 2026-04-17
authors:
  - Claude Code (council: devils-advocate + architect-advisor)
scope: Evaluation of the shipped Go acpmock solution vs. the planned @copilotkit/aimock approach
---

# E2E Mock Strategy — Post-Implementation Analysis

## Purpose

The E2E initiative (14 tasks, all completed) originally planned `@copilotkit/aimock` as a Node-based mock for deterministic assistant streaming. The shipped implementation replaced this with a pure-Go ACP mock driver. This document answers three questions:

1. Why was `@copilotkit/aimock` dropped?
2. Is the Go solution equivalent, weaker, or stronger than the aimock plan would have been?
3. Does the current E2E suite produce reliable confidence, and if not, what breaks?

Input sources: `.compozy/tasks/e2e/_techspec.md`, `adrs/adr-001.md`..`adr-005.md`, `task_01.md`..`task_14.md`, `memory/MEMORY.md`, and the shipped code under `internal/testutil/acpmock/`, `internal/testutil/e2e/`, `internal/daemon/*_integration_test.go`, `internal/e2elane/`, and `web/e2e/`.

## TL;DR

The Go acpmock is **architecturally sound and materially equivalent to — or better than — the original aimock plan**. The aimock-vs-Go debate is a red herring. The real problem is that the current suite has three independent weaknesses that **neither approach would have fixed**: stale ADR/techspec documentation, substring-based prompt matching, and happy-path-only fixture coverage. All three can be fixed with ~2 days of surgical work; none of them require touching the Node vs. Go decision.

The web E2E lane does **not** use aimock because it has no surface to mock. Playwright drives the real daemon subprocess; agent behavior is produced by the same Go acpmock used by the runtime lane.

---

## 1. What shipped vs. what was planned

### Planned (ADR-001, _techspec.md, task_02.md)

- Test-only Node/TypeScript subprocess: `node <abs>/internal/testutil/acpmock/driver/dist/index.js --fixture <path>`
- `@copilotkit/aimock` for deterministic assistant text chunking and stream cadence
- `mock-acp` package with its own bundling/dist pipeline

### Actually shipped

- Pure-Go ACP mock driver: `internal/testutil/acpmock/cmd/acpmock-driver/main.go`
- Uses `github.com/coder/acp-go-sdk` directly (same SDK as the real daemon)
- Compiled on-demand: `internal/testutil/acpmock/driver_binary.go:36-65` runs `go build` to a temp path at first test use
- Fixture schema in `fixture.go` (JSON, six `StepKind` values): `assistant`, `thought`, `tool_call`, `permission`, `sandbox_exec`, `bridge_response`
- Eight fixture files in `testdata/`
- Node/`dist/` seam fully deleted from disk; `@copilotkit/aimock` is not in any `package.json`

There is **no ADR superseding ADR-001**. The implementation diverged from the accepted decision record, and the doc corpus still describes the Node plan as if it were real.

---

## 2. Why the migration happened (inferred)

No explicit rationale was documented. The reconstructable reasons from ADR-001 risks, MEMORY.md shared learnings, and the shape of the shipped code:

- ADR-001 already flagged cross-platform Node rendering as a risk.
- `aimock` scope was narrow from the start ("assistant text chunking and stream cadence only"). A ~10-line Go loop (`main.go:241-260`) reproduces it.
- `coder/acp-go-sdk` gives AGH a first-class Go ACP implementation. Using it means the mock driver and the real daemon share types. No TS-side encoder, no parity tax.
- Keeping the toolchain Go-only removes Node bootstrapping from CI for test-only workflows.

The decision is defensible. The problem is that it was never written down.

---

## 3. Does the web lane justify aimock? No.

The common intuition — "the web is TypeScript, aimock would fit there" — does not survive inspection.

### Where aimock could plug in theoretically

1. **Inside an ACP subprocess written in Node.** This was the original ADR-001 plan. When the driver became Go, this insertion point disappeared.
2. **Inside a web component that calls an LLM directly.** AGH has no such component. `grep` over `web/src` for `@copilotkit`, `aimock`, `MockLanguageModel`, `generateText`, `streamText` returns nothing that represents a browser-side LLM call. `useChat` from `@ai-sdk/react` is used in `web/src/systems/session/hooks/use-session-chat.ts:2` purely as an SSE consumer — the transport, not the model.

### Where agent behavior actually originates in web E2E

`web/e2e/session-onboarding.spec.ts:7-16` points at `internal/testutil/acpmock/testdata/browser_session_lifecycle_fixture.json` — the **same Go fixture** consumed by the **same Go driver**. `web/e2e/fixtures/runtime.ts:298-306` literally does `go build -o ... ./cmd/agh` and spawns the real daemon as a subprocess. Playwright then drives the browser against it.

Conclusion: the web E2E is a browser-interaction lane, not an agent-behavior lane. Introducing aimock here would mean reverting `task_02` (reintroduce a Node ACP driver) to get the same fixture semantics the Go driver already delivers. Cost real, benefit zero.

---

## 4. Strengths of the current solution

Both advisors converged on these:

1. **Shared SDK means zero protocol drift.** Mock driver and daemon both use `coder/acp-go-sdk`. An aimock-based Node driver would have required a second TypeScript ACP encoder and imposed continuous parity checks on every SDK bump.
2. **`cmd/acpmock-driver` is properly isolated.** Under `internal/testutil/`, only compiled via `exec.Command("go", "build", ..., "./internal/testutil/acpmock/cmd/acpmock-driver")` (`driver_binary.go:42-49`). No production file imports it. Release builds and `go build ./cmd/agh` are unaffected.
3. **Registration through real agent definitions.** `registration.go:85-97` writes a real AGENT.md, then round-trips it through `aghconfig.LoadAgentDefFile` and `ResolveAgent`. The mock goes through the same validation pipeline as any production agent. No test-only flags in production code, no driver-injection seams.
4. **One mock, two lanes.** Runtime E2E and browser E2E share the same fixtures and driver. A single place to change agent behavior.

---

## 5. Weaknesses of the current solution

### 5.1 Documentation-vs-code drift is load-bearing

| Location | Stale claim | Reality |
|---|---|---|
| `adrs/adr-001.md:24,66` | Mandates `node <abs>/internal/testutil/acpmock/driver/dist/index.js` | No Node driver, no `dist/` |
| `_techspec.md:23,192-194` | Lists `@copilotkit/aimock` as Integration Point | Not a dependency anywhere |
| `task_02.md:14,53` | Describes Node driver workspace and aimock scope | Go driver under `cmd/acpmock-driver` |
| `reviews-001/issue_001.md`..`issue_018.md` | Blame missing `driver/dist/index.js` for build failures | File never existed in this tree |

Any new contributor reading the ADRs first will reintroduce a Node driver. This is not a cosmetic issue — ADR-001 is marked *Accepted* and has not been superseded.

### 5.2 Prompt matching is substring-on-stringified-frame

`internal/testutil/acpmock/fixture.go:227-240` uses `equals`/`contains` substring matching against the ACP prompt payload (`extractPromptText` in `main.go:718-730`).

Fixtures like `network_collaboration_fixture.json:34-36` depend on substrings (`kind="say"`, `kind="direct"`) that are implicit output of the daemon's prompt-rendering template. Any refactor to how network envelopes are serialized into an assistant's prompt can either:

- cause "no turn matched" (loud failure — recoverable)
- silently route the turn to a different fixture entry, producing **green tests that assert against phantom state** (silent false negative — critical)

There is no declared contract between daemon prompt construction and fixture matchers. They are coupled through untyped strings. At 2–3x the current fixture volume this becomes a significant foot-gun.

### 5.3 Coverage is happy-path only

Eight fixture files model a cooperative, honest agent. Zero fixtures for:

- Mid-stream crash / OOM kill
- Malformed tool-call JSON
- Permission-request followed by disconnect
- `LoadSession` with divergent prompt history
- Concurrent prompts on one session
- Partial stream with backpressure
- Half-close framing

`main.go:104-106` stubs `Authenticate` to always succeed; `SetSessionMode`/`SetSessionModel` at `main.go:146-158` are no-ops. Real Claude Code / Codex / Gemini subprocesses are not cooperative — they crash, stall, emit tool calls that do not round-trip the schema, and get killed by OOM. None of that surface is exercised.

"We test the daemon, not the agent" is a legitimate scope. But calling the result "E2E confidence" while the test double is a cooperative idealization is **marketing, not engineering**.

### 5.4 Over-determinism loses real concurrency signal

`main.go:785-795` pauses a fixed 5ms between every `SessionUpdate`. Real ACP subprocesses have non-deterministic stdio buffering, partial JSON-RPC framing, and stream backpressure. A Node driver doing real line-buffered writes would have been **more** likely to shake loose race conditions in `internal/acp` and `internal/session` stream consumers.

This suite cannot catch ACP framing bugs, stdio half-close ordering bugs, or backpressure deadlocks. This is not a claim that aimock would have been better — it is a claim that the current driver is **too tame**.

### 5.5 On-demand `go build` in tests has no pinning, cache, or concurrency protection

`driver_binary.go:36-65` calls `os.MkdirTemp` + `go build` the first time each test binary runs. Different packages running in parallel under `go test ./...` each shell out to `go build`, each produce a separate temp binary, none are cleaned up. The `driverBinaryPath` cache is process-local — useless across test binaries.

On a cold CI runner with module cache miss, this fans out into N parallel `go build` invocations racing the same `GOMODCACHE`. Flake vector, disk-fill vector, and slow-CI vector in one. A prebuilt `dist/` approach would have avoided this specific class of issue.

### 5.6 Fixture DSL is accreting without a factoring plan

`fixture.go:16-23` already has six `StepKind` values. Each new domain concern — cancel mid-stream, server-initiated events, partial tool output, interleaved thought/assistant, progress updates, network-origin turn deltas — is another kind. `Step` is a flat struct with mutually-exclusive optional fields (e.g., `Command`/`Args` only apply to `StepKindEnvironment`).

`FixtureVersion = 1` is static with no upgrade plan. At six kinds this is cheap to refactor. At fifteen kinds it becomes a migration.

### 5.7 Narrow waist is present but undocumented

`internal/testutil/e2e/mock_agents.go` exposes `RegisterMockAgent(spec MockAgentSpec)`. This is the correct narrow waist for replacing the mock implementation later. But it is not documented as a required boundary, and the 7+ consumers in `daemon_*_integration_test.go` are not audited for direct `acpmock.Register` usage. If that interface holds, migrating to aimock, VCR, or property-based fuzzing later is a rewrite of one package, not fourteen.

---

## 6. Verdict on aimock equivalence

| Dimension | aimock (hypothetical) | Go driver (shipped) |
|---|---|---|
| Deterministic chunking | Equal | Equal |
| Streaming cadence | Slightly more realistic (Node I/O) | Fixed 5ms pauses |
| Protocol drift | Real risk (separate TS encoder) | Zero (shared SDK) |
| CI stability (Node runtime + bundling) | Adds Node + dist pipeline | Go-only |
| Race-condition coverage | Slightly better | Weaker |
| Real-LLM bug coverage | **Zero** | **Zero** |
| Tool-call schema drift coverage | **Zero** | **Zero** |

The Go driver is **not less robust than aimock would have been**. It is equal-or-better on architectural coupling, roughly equivalent on determinism, and weaker only on one specific axis (concurrency surface). The confidence ceiling is identical because both are scripted doubles.

**Neither approach produces "real LLM confidence" — that was never part of the plan.** If the team wants real LLM coverage, it needs a new lane (credentialed, opt-in, nightly) running 1–2 canonical flows against real providers. That is a separate initiative from the aimock-vs-Go question.

---

## 7. Recommendations

Ordered by blast radius, highest first.

### P0 — Documentation reconciliation (1–2 hours)

1. Write **ADR-006: ACP Mock Is Implemented in Go**, marking ADR-001 as *Superseded*. Include rationale: shared SDK, no cross-language parity tax, Go-only CI.
2. Rewrite `_techspec.md:23,192-194` to remove `@copilotkit/aimock` as an Integration Point.
3. Rewrite `task_02.md:14,53` to describe the shipped Go driver path.
4. Audit and close `reviews-001/issue_001.md`..`issue_018.md` entries that reference the phantom `driver/dist/index.js`.
5. Remove any empty `driver/` directory remnants on disk if they still exist.

### P1 — Structural hardening (1–2 days)

6. Replace substring prompt matching with a structured matcher. Introduce a new `StepMatch` type that takes typed envelope fields (`envelope_kind`, `from`, `to`, `prompt_hash`) that the daemon's prompt serializer populates explicitly. Keep `contains`/`equals` as fallback for legacy fixtures during migration.
7. Freeze the `StepKind` set at six for fixture v1. Document that new protocol behaviors either (a) land as a generic `custom` step with a typed payload, or (b) require `FixtureVersion = 2` with a documented migration.
8. Add three "misbehaving agent" fixtures: crash-mid-stream, malformed tool-call JSON, permission-request-then-disconnect. Assert the daemon surfaces the failure cleanly on public surfaces.
9. Document `MockAgentSpec` in `internal/testutil/e2e/mock_agents.go` as the required narrow-waist API. Audit daemon integration tests to confirm no direct `acpmock.Register` usage.

### P2 — CI hygiene (half day)

10. Build the `acpmock-driver` binary once per `go test ./...` invocation via a `TestMain` in a shared test helper, cache the path by content-hash of the source, and reuse across test binaries. Eliminates the parallel `go build` race.
11. Add CI step `go vet ./internal/testutil/acpmock/...` and `go test -race ./internal/testutil/acpmock/...` to the lint gate.

### P3 — Explicit boundary on LLM coverage (half day)

12. Write **ADR-007: No Lane Exercises Real LLM Providers**. State the rationale (LLM quality is a product concern, not an integration concern; cost and flake ceiling of credentialed lanes). Mark explicitly: if a future initiative wants real-LLM coverage, it belongs in a new tier, not in acpmock.

---

## 8. What the current suite actually guarantees

For honesty with stakeholders, this is what the PR-required + nightly E2E suite proves today, and what it does not:

### Proves
- Daemon boot, config loading, workspace resolution, SQLite schema, embedded NATS, HTTP/UDS/CLI transports
- ACP JSON-RPC subprocess boundary (for a cooperative agent)
- Network RFC correlation for scripted flows
- Automation webhook, manual trigger, and task-backed delegation for happy paths
- Bridge ingress, route creation, delivery broker progression (with scripted agent replies)
- Environment sandbox allow/block on known operations
- Browser UI journey completion on daemon-served assets
- Artifact capture on failure

### Does not prove
- Daemon behavior when the ACP subprocess crashes, stalls, or emits protocol violations
- Tool-call round-trip correctness against real provider schemas (Claude, OpenAI, Gemini)
- Context-window truncation, token budget exhaustion, model refusal
- Streaming backpressure and half-close framing
- Any interaction with real LLM latency, retry, or rate-limit behavior
- Performance regressions under realistic agent output volume

The gap between these lists is the honest scope of "E2E confidence" the current suite delivers.

---

## 9. Council attribution

The analysis in sections 4, 5, and 6 synthesizes independent evaluations from two advisory agents:

- **The Devil's Advocate** — attacked the equivalence claim and the "reliable E2E" framing. Found sections 5.1, 5.2, 5.3, 5.4, 5.5 independently.
- **The Architect** — evaluated long-term structural soundness, DSL evolution, and migration cost. Found sections 4, 5.1, 5.2, 5.6, 5.7 independently.

Both converged on the same three primary weaknesses (5.1, 5.2, 5.3) without coordinating. That convergence is the strongest signal that these are the real problems, not Node-vs-Go.

## 10. File reference index

### Shipped implementation
- `internal/testutil/acpmock/cmd/acpmock-driver/main.go` — driver entry
- `internal/testutil/acpmock/fixture.go` — schema + validation
- `internal/testutil/acpmock/registration.go` — AGENT.md rendering
- `internal/testutil/acpmock/driver_binary.go` — on-demand build
- `internal/testutil/acpmock/testdata/*.json` — eight fixtures
- `internal/testutil/e2e/mock_agents.go` — narrow waist for consumers
- `internal/e2elane/lanes.go` — test lane matrix
- `web/e2e/fixtures/runtime.ts` — browser harness that spawns the Go daemon
- `web/e2e/session-onboarding.spec.ts` — reference browser spec

### Stale documentation
- `.compozy/tasks/e2e/adrs/adr-001.md` — needs supersession
- `.compozy/tasks/e2e/_techspec.md` (lines 23, 192-194) — needs rewrite
- `.compozy/tasks/e2e/task_02.md` (lines 14, 53) — needs rewrite
- `.compozy/tasks/e2e/reviews-001/issue_001.md`..`issue_018.md` — needs audit
