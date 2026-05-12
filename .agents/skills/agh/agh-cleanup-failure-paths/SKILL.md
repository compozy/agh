---
name: agh-cleanup-failure-paths
description: >-
  Audits Go functions with multi-step setup or teardown for cleanup discipline on
  every error return: cancel any context created or extended, close every opened
  resource, release every claim or lease, stop every spawned subprocess, drain
  every HTTP body. Forbids http.DefaultClient in production code paths. Use when
  editing functions in internal/acp, internal/session, internal/scheduler,
  internal/coordinator, internal/extension, internal/automation, internal/store,
  internal/memory, internal/api, or any subprocess or registry handler. Do not
  use for read-only helpers, pure data structures, or tests.
trigger: implicit
---

# Cleanup Failure Paths

Hermes review issue #001 cost a real PR round because a `procCtx` leaked when registry registration failed. The happy path was clean; the partial-failure path leaked. This pattern recurs in every AGH PR that adds multi-step setup. Activate this skill before editing any function that creates/extends a context, registers a resource, opens a connection, or spawns a subprocess.

## Procedures

**Step 1: Identify Setup-Step Boundaries**

1. Read the target function and enumerate every "step" that allocates a resource: context creation, file open, listener bind, registry register, lease claim, goroutine spawn, subprocess start, HTTP request, mutex lock, transaction begin.
2. For each step, identify the cleanup action that pairs with it: `cancel()`, `Close()`, `Unregister()`, `Release()`, `Stop()`, `defer cancel()`, `tx.Rollback()`.
3. Read `references/cleanup-table.md` for the canonical pairing table.

**Step 2: Walk Every Error Return**

1. List every `return ... err` statement in the function.
2. For each one, walk backward through the function body and confirm every resource allocated above the return point is cleaned up — either by an immediate `defer` adjacent to its allocation OR by an explicit cleanup call before the return.
3. If a resource is allocated and the only cleanup is on the success path, that's a leak. Add cleanup on the error path.
4. Treat panic recovery and `runtime.Goexit` as additional exit paths — `defer` covers both.

**Step 3: Apply the Defer-Adjacent Rule**

1. Pair each `WithCancel`/`WithTimeout`/`WithDeadline` with a `defer cancel()` on the next line.
2. Pair each `os.Open`/`net.Listen`/`db.Begin`/`tx.Begin` with a `defer Close()`/`defer tx.Rollback()` on the next line.
3. Goroutine spawns paired with WaitGroup increment and a `defer wg.Done()` inside the goroutine.
4. Subprocess starts paired with a deferred `Stop()` whose context respects `select { case <-proc.Done(): case <-ctx.Done(): }`.

**Step 4: Detached-Lifetime Discipline**

1. Any work that outlives the request (prompt execution, channel send, automation job) MUST detach via `context.WithoutCancel(ctx)`.
2. The writer loop stays bound to the request context — detach the *execution*, not the *response*.
3. `context.WithoutCancel` does NOT preserve deadlines — re-attach with `WithDeadline` if needed.
4. Expose explicit cancel endpoints for detached work.

**Step 5: Outbound-Call Hygiene**

1. `http.DefaultClient` is forbidden in production paths. Every outbound HTTP call uses an explicit timeout.
2. Drain response bodies via `io.Copy(io.Discard, resp.Body)` then `resp.Body.Close()` — do not skip the drain.
3. For TLS-sensitive endpoints, require HTTPS and OAuth/PKCE per `docs/_memory/lessons/L-008-schema-migrations-mandatory.md` (no, that's a sibling lesson — see Hermes review issues 015/016/017 for OAuth discovery, RFC 8414 well-known URLs, and HTTPS enforcement).

**Step 6: Test the Failure Paths**

1. For each error return identified in Step 2, write or extend a test that triggers that exact failure and asserts the cleanup happened (resource released, context cancelled, lease unblocked, subprocess reaped).
2. Read `references/test-failure-paths.md` for canonical patterns.
3. Mocking via interfaces is preferred over runtime fault injection.

## Error Handling

- **Function is too large to audit in one pass:** break the audit at the function boundary; audit only the function being edited. Cross-function cleanup is the caller's responsibility.
- **Cleanup requires a resource not in scope:** the function is structured wrong — push the cleanup responsibility up to the caller via an opener-closer pair, or restructure into a helper that returns a `cleanup func()` callback.
- **Existing function lacks any error-path cleanup:** flag it as pre-existing technical debt; add cleanup for the path you're editing and note the rest in the task body for follow-up.
- **`defer` count exceeds reasonable bounds (e.g., >6 in one function):** the function is doing too much. Recommend splitting before adding more defers.
