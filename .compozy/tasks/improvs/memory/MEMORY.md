# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Package-level improvements tasks require report-first execution: inventories and benchmark baselines must be written before fixes.

## Shared Decisions
- If the environment does not expose a concrete UBS skill runner, record `ubs` as `not-run` with the literal tooling limitation rather than substituting a manual review.
- Repository commitlint currently rejects scoped commit headers (`scope must be empty`), so task commits must use unscoped subjects like `refactor: <package> improvements pass`.
- Package improvements tasks still need local task/task-list status updates, but those tracking-only files should stay out of the automatic deliverable commit unless a task explicitly requires otherwise.

## Shared Learnings
- Workspace-controlled `.env` values in `internal/config` must stay scoped to the active load; mutating process env leaks `AGH_HOME` and automation webhook-secret resolution across later workspace loads.
- The repository's `errcheck` configuration flags single-value generic type assertions in helper code, so generic dispatch helpers should prefer checked interface assertions or non-generic casts.
- Package-task `make verify` runs on this macOS setup consistently emit non-blocking environment warnings (`NO_COLOR` ignored because `FORCE_COLOR` is set, and `ld: warning: -bind_at_load is deprecated on macOS` from the vendored `golangci-lint` build), so reports should record them separately from the exit-0 verification result rather than treating them as package-local regressions.
- For improvements-task benchmark decisions, use medians from the exact full before/after command recorded in the report (`go test -bench=. -benchmem -count=5 ./internal/<pkg>/...`). Targeted reruns are useful for local diagnosis but should not drive the final fixed/not-hot verdict when they disagree with the full-suite command.
- In very small Go packages, new test/benchmark scaffolding can dominate the duplication scan; collapse repeated constructor matrices into shared helpers or suites before finalizing the report so the remaining duplication signal reflects real production tradeoffs.
- For scheduler-style packages, never invoke user-provided or re-entrant sinks while holding the core state mutex; capture events under lock and emit them after unlock to avoid self-deadlocks.
- When a package keeps a registry of per-key mutexes, track active references and prune idle entries instead of leaving the registry to grow for the process lifetime.
- For benchmark-driven file-hash optimizations in Go, `io.CopyBuffer` may not reduce allocations when the reader has a fast path; if the allocation profile still regresses, switch to an explicit reusable-buffer `Read` loop and re-benchmark before keeping the change.
- For hot `json.RawMessage` helpers in Go, prefer `bytes.TrimSpace` and `bytes.Equal` over string conversions; string-based trimming/comparison introduces avoidable allocations on every call.
- When a Go transcript/normalization helper receives tool output that is already decoded as `map[string]any`, reuse that map instead of re-marshaling and re-unmarshaling it; only decode `json.RawMessage` values when the trimmed payload is object-shaped.
- `internal/cli/client.go` still keeps a duplicated SSE decoder instead of calling `internal/sse.Decode`, so any future scope touching CLI streaming should reconcile the two implementations to avoid behavior drift.
- Store-layer scan helpers should treat malformed persisted timestamps as data-corruption errors and return wrapped parse failures instead of silently dropping invalid timestamp fields.
- In subprocess-style monitor loops, per-probe timeout contexts must derive from the parent lifecycle context; using `context.Background()` inside the probe can stall shutdown even when the outer goroutine already watches `ctx.Done()`.
- For subprocess transports that multiplex child responses and child-originated requests on the same stdout stream, naive request backpressure can deadlock response handling if queued requests block the reader ahead of pending responses; any concurrency cap needs a transport-level design, not a local semaphore patch.
- Test helpers that are intentionally used inside `t.Cleanup` callbacks must not be blindly re-rooted onto `testing.TB.Context()`: that context is canceled before cleanup begins, so teardown operations inherit immediate cancellation. Keep cleanup-usable helper contexts background-rooted and add a cleanup-time regression test before changing that contract.
- For eager register-and-resolve flows, compensating cleanup after a successful insert must not reuse a caller context that may already be canceled; use `context.WithoutCancel(ctx)` (or an equivalent cleanup context) for rollback deletes so partial registrations do not persist on cancellation paths.

## Open Risks
- UBS invocation may be blocked for all package tasks unless a real skill runner is discovered later in the run.

## Handoffs
