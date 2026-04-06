# Remove Legacy/Compat Runtime Cleanup

## Summary

- Keep boot-time `Reconcile` because it is current crash-recovery logic for repairing `agh.db` against `~/.agh/sessions/`, not legacy support.
- Remove all legacy session-metadata compatibility code and make runtime packages strict about current-schema data and valid internal wiring.
- Apply the broader sweep only to the same anti-patterns in the same runtime/session boundary; do not turn this into an unrelated repo-wide cleanup.

## Key Changes

- In `internal/observe/reconcile.go`, delete `legacySessionMeta` and `readLegacyStoppedSessionMeta()`. Change session-dir scanning to accept only current `store.SessionMeta`; if a `meta.json` is malformed or old-format, skip that directory and emit a warning instead of treating it as supported input.
- In `observe/`, remove the `store` alias re-exports from `query.go` and update `Observer`, `daemon`, `httpapi`, `udsapi`, and tests to use concrete `store.EventSummary`, `store.EventSummaryQuery`, `store.TokenStats`, `store.TokenStatsQuery`, `store.PermissionLogEntry`, and `store.PermissionLogQuery` directly.
- In `internal/session/interfaces.go`, remove the `acp` alias re-exports and migrate all internal consumers to concrete `acp.StartOpts`, `acp.PromptRequest`, `acp.ApproveRequest`, `acp.AgentEvent`, `acp.ACPCaps`, and `acp.TokenUsage`.
- Remove nil-receiver and nil-context fallback behavior from `observe.Observer` methods and from the thin runtime wrappers in `session/interfaces.go`. Treat those as programmer errors instead of supported alternate paths. Apply the same stricter cleanup to the matching `internal/acp.AgentProcess` helper methods discovered in the broader sweep.
- Remove `nopNotifier` from production code. Make notifier dispatch explicitly optional in `session.Manager` with a `nil` check at the call sites, and move any no-op notifier behavior needed by tests into `_test.go` test doubles.
- Remove `nopServer` from production code in `internal/daemon/daemon.go`. Replace it with daemon test-local server doubles and keep the production `Server` surface strict.
- Remove the duplicated `firstNonBlank` helpers without introducing a new generic utility. Replace each usage with intent-specific logic:
  - `config`: explicit trimmed command/model resolution
  - `acp`: direct turn-id merge logic
  - `httpapi`: explicit tool-name/error-text fallback logic
- Remove the duplicated `cloneStringMap` helpers by inlining the map copy at the two payload-construction call sites. Do not create a new shared package only to hold this helper.

## Interface Changes

- `observe` query-facing interfaces will use `store` types directly instead of `observe` aliases.
- `session` and downstream interfaces will use `acp` types directly instead of `session` aliases.
- `session.NewManager` will no longer rely on a built-in production no-op notifier.
- Nil `Observer`, nil `AgentProcess`, and nil `ACPDriverAdapter` behavior will no longer be treated as supported contracts.

## Test Plan

- Replace the legacy reconcile test with current-schema-only coverage: valid metadata indexes, malformed/old metadata is skipped and warned, missing directories still orphan, and recovered current-schema metadata still normalizes to `stopped`.
- Remove tests that assert nil-observer, nil-agent-process, nil-adapter, `nopNotifier`, or production `nopServer` behavior. Replace them with tests for the new explicit contracts and test-local doubles.
- Update interface doubles across `internal/observe`, `internal/session`, `internal/acp`, `internal/daemon`, `internal/httpapi`, `internal/udsapi`, and `internal/cli` to import `store` and `acp` concrete types directly.
- Verification order: targeted package tests for the touched runtime packages first, then `make verify` as the completion gate.

## Assumptions

- Malformed or old session metadata is not supported in alpha; the system should ignore it with visibility, not preserve compatibility.
- `Reconcile` stays because it repairs current crash inconsistencies between session directories and the global DB.
- Broader sweep means adjacent same-class anti-patterns in the runtime/session boundary, not an open-ended architecture rewrite.
