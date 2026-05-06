---
provider: manual
pr:
round: 1
round_created_at: 2026-05-06T02:21:18Z
status: resolved
file: internal/daemon/native_tools.go
line: 1511
severity: high
author: claude-code
provider_ref:
---

# Issue 002: Sub-agent native memory writes are not denied

## Review Comment

Safety invariant 6 in the TechSpec is explicit:

> "Sub-agent read-only. Sub-agent controllers are configured with `Mode = ReadOnly`; write tools fail closed. Defense in depth: native tool registry also denies writes for `actor_kind == agent_subagent`. Parent-side extractor may process sub-agent traces, but every resulting Decision carries `Provenance.SourceActor = "agent_subagent"`."
> (`.compozy/tasks/mem-v2/_techspec.md` §Safety Invariants)

Neither layer of defense is wired:

1. **Tool registry has no actor-kind check.** `daemonNativeTools.memoryPropose` (`internal/daemon/native_tools.go:1511-1576`) and `memoryNote` (`internal/daemon/native_tools.go:1578-1619`) decode input, resolve the write store, and call `location.Store.ProposeWrite` / `ProposeDelete` without ever inspecting the caller's actor kind. The dispatch payload `tools.Scope` (`internal/tools/tool.go:409-414`) and `tools.CallRequest` (`internal/tools/tool.go:430-441`) carry `WorkspaceID`, `SessionID`, `AgentName`, `Operator`, but no `ActorKind` field — there is no way for the handler to detect a sub-agent caller even if it tried.

2. **Controller `Mode = ReadOnly` is set but never consulted.** `FrozenSnapshot.ControllerMode` is correctly flipped to `SnapshotControllerReadOnly` for inherited sub-agent snapshots (`internal/memory/snapshot.go:187`, plus the `controllerModeForSession` helper at line 510). However, a project-wide search shows the only callers of `SnapshotControllerReadOnly` are the constant definition and one assertion in `assembler_test.go:342`. Nothing in `internal/memory/controller`, `internal/memory/decision.go`, `internal/memory/extractor`, or `internal/daemon` reads `snapshot.ControllerMode` to fail-closed on writes.

The combined effect: a sub-agent that runs inside a session with the `agh__memory_propose` or `agh__memory_note` tool registered can mutate durable memory directly, bypassing the parent's extractor pipeline. Any decision written that way is attributed via `Origin = OriginTool` (`internal/daemon/native_tools.go:1571,1614`), so audit downstream records the write as the user — exactly the misattribution the spec calls out:

> "every resulting Decision carries `Provenance.SourceActor = 'agent_subagent'` so synthesized entries are not attributed to the user."

The QA verification report (`.compozy/tasks/mem-v2/qa/verification-report.md`) does not mention a sub-agent-write probe, and `internal/memory/extractor/runtime_test.go:29` only verifies sub-agent **extractions** are skipped, not sub-agent **direct tool writes**.

Suggested fix:

- Plumb `actor_kind` (or at least an `IsSubAgent` boolean derived from the session's actor classification) into `tools.Scope` or `tools.CallRequest` so the daemon dispatcher can pass it to native handlers. The session/spawn metadata already tracks `actor_kind` via `internal/session/spawn.go`; expose it on the dispatch boundary.
- In `memoryPropose` and `memoryNote`, fail-closed when the resolved actor kind is `agent_subagent`, returning `toolspkg.ErrToolPermissionDenied` (or a new `memory.subagent_write_denied` deterministic error code).
- Optionally consult `FrozenSnapshot.ControllerMode` from the session's captured snapshot as the second defensive layer, since the spec explicitly calls out two layers ("Defense in depth").
- Add tests asserting:
  - `agh__memory_propose` from `actor_kind = agent_subagent` returns the deterministic deny error and emits `memory.write.rejected` with `reason = "subagent_write_denied"`.
  - `agh__memory_note` follows the same deny path.
  - Root-agent calls remain successful (regression).

## Triage

- Decision: `VALID`
- Root cause: native memory mutation tools had no trusted caller classification at dispatch time, and `memoryPropose`/`memoryNote` never performed a second fail-closed check for sub-agent callers. Projection policy could hide write tools from normal sub-agent sessions, but a direct call path could still reach the handlers.
- Fix approach: carry `actor_kind` through `tools.Scope`/`CallRequest`, derive it from session lineage when absent, deny `agent_subagent` writes in the native handlers with deterministic `memory_subagent_write_denied`, emit `memory.write.rejected`, and record successful root tool writes for extractor mutual exclusion.

## Resolution

- Plumbed `actor_kind` through tool scope/request normalization and added deterministic native-tool denial for `agent_subagent` calls to `agh__memory_propose` and `agh__memory_note`.
- Rejection now emits `memory.write.rejected` with `memory_subagent_write_denied`; successful root tool writes are recorded for extractor mutual exclusion.
- Verification: `go test ./internal/daemon -run 'TestDaemonNativeTools' -count=1` passed; `go test ./internal/tools ./internal/cli -count=1` passed; `make verify` passed with Bun 334 files / 2150 tests, Go `DONE 8393 tests in 90.274s`, and boundaries OK.
