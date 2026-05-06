---
provider: manual
pr:
round: 1
round_created_at: 2026-05-06T02:21:18Z
status: resolved
file: internal/memory/extractor/runtime.go
line: 158
severity: medium
author: claude-code
provider_ref:
---

# Issue 005: Extractor lacks mutual-exclusion with explicit memory_propose

## Review Comment

ADR-010 / TechSpec Extractor section requires the extractor to no-op for any turn where the agent already proposed memory directly (`.compozy/tasks/mem-v2/_techspec.md` §Extractor (Mode A — ADR-010)):

> "Mutual exclusion. If main agent invoked `agh__memory_propose` in the same turn, extractor no-ops for that turn (`hasMemoryWritesSince` analog)."

`Runtime.HandleSessionMessagePersisted` (`internal/memory/extractor/runtime.go:158-205`) only checks two gates:

1. Drop dream/system session types (`runtime.go:168-171`).
2. Drop sub-agent or non-root callers (`runtime.go:172-174`).

There is no detection of whether the same turn already produced a controller decision (e.g., a `memory_decisions` row whose `actor_kind = "agent_root"` and whose `until_message_seq` matches the persisted turn). A workspace-wide search confirms the absence: `grep -r "hasMemoryWritesSince\|HasMemoryWrites\|extractorMutualExclusion"` returns no matches anywhere in `internal/`.

Operationally this means a turn where the agent calls `agh__memory_propose` is followed by the extractor sub-agent re-deriving candidates from the same transcript. The controller's exact-content-hash NOOP rule (`controller.go:122-133`) deduplicates the disk write, but:

- The extractor still spends LLM tokens and time on the duplicate extraction (cost envelope OC8 in §Open Concerns).
- The duplicate extraction enters the bounded coalesce queue (`runtime.go:243-258`), increasing the chance that real new turns get dropped due to `coalesceMax = 16`.
- The audit ledger gets two parallel records — one decision committed via `OriginTool`, another (NOOP/REJECT) via `OriginExtractor` — for the same logical fact, complicating forensics.

Suggested fix:

- Plumb a "has tool-driven write since seq N" probe into the extractor entry point. Cheapest implementation: track the latest `memory_decisions.id` written via `OriginTool` per session in memory (the daemon already passes `memcontract.OriginTool` at `native_tools.go:1571,1614`), and short-circuit `Enqueue` when the in-flight turn's `since_message_seq..until_message_seq` overlaps the recorded tool-write seq.
- Alternatively, add a `Store.SessionToolWritesSince(ctx, sessionID, sinceSeq)` query backed by `memory_decisions WHERE actor_kind = 'agent_root' AND origin = 'tool' AND since_seq >= sinceSeq` and gate `Runtime.Enqueue` on its result.
- Test coverage that should land with the fix:
  - `TestRuntime_NoopsWhenToolWriteOccurredInSameTurn` (matches `TestExtractor_MutualExclusionWithProposeTool` already enumerated in §Test Plan §Extractor).
  - Negative case: regular root turn with no tool writes still extracts.
  - Boundary case: tool write in turn N-1 does NOT suppress extraction for turn N.

The current implementation effectively breaks ADR-010 §Mode A's mutual-exclusion property even though the surrounding queue/coalesce/DLQ machinery is in place.

## Triage

- Decision: `VALID`
- Root cause: the extractor runtime only filtered dream/system/sub-agent turns and had no same-turn marker for successful explicit root memory tool writes. A root turn using `agh__memory_propose` or `agh__memory_note` could therefore enqueue extractor work for the same persisted message.
- Fix approach: record root memory tool writes in the daemon-owned extractor runtime and consume the marker from `HandleSessionMessagePersisted` so the matching turn no-ops while later turns still extract normally. Cover exact-turn, next-turn, and marker-without-sequence behavior in runtime tests.

## Resolution

- Added root tool-write markers to the daemon-owned extractor runtime and consume them from `HandleSessionMessagePersisted` so same-turn explicit memory writes suppress extractor work without suppressing later turns.
- Covered exact sequence, stale marker, and sequence-less marker behavior in extractor runtime tests.
- Verification: `go test ./internal/memory/extractor -count=1` passed; `go test -race ./internal/memory/recall ./internal/memory ./internal/memory/extractor ./internal/api/core ./internal/daemon ./internal/tools ./internal/cli -count=1` passed; `make verify` passed with Bun 334 files / 2150 tests, Go `DONE 8393 tests in 90.274s`, and boundaries OK.
