# Implementation Peer Review Round 1 Summary

- Verdict: `FIX_BEFORE_SHIP`
- Blockers: 1
- Risks: 3
- Nits: 4

## Blockers

- `B-001` — `web/src/systems/network/components/direct-room.tsx:86`: missing-direct error copy still uses `AGH could not load ...` while the matching missing-thread copy was corrected in round 001. Fix by changing the direct-room copy to `Could not load direct room ${directId}. Choose an existing direct room from #${channel}.`, updating any matching test expectation, and rerunning `make verify`.

## Risks

- `R-001` — `internal/store/sessiondb/session_db.go:350`: `Query` / `QueryHookRuns` hold `acceptMu.RLock()` through SQL query and row scan, changing shutdown latency semantics for `Close()`.
- `R-002` — `internal/session/query.go:167`: second `waitForSessionFinalization` branch lacks a comment and direct behavior coverage.
- `R-003` — `web/src/systems/network/hooks/use-direct-room.ts:46`: `isMessagesLoading` no longer reflects detail-resolution loading for future consumers; same concern applies to `use-thread-overlay.ts:48`.

## Nits

- `N-001` — `internal/session/query_test.go:477`: subtest name overstates the assertion.
- `N-002` — `internal/session/query_test.go:486`: cleanup uses `_ = h.manager.Stop(...)` without a written justification.
- `N-003` — `internal/store/sessiondb/session_db_extra_test.go:42`: new post-close `Query` assertion is not in its own `t.Run("Should ...")` subtest.
- `N-004` — `web/src/systems/network/components/empty-states/conversation-error.tsx:16`: default `testId` appears unused because all call sites pass an explicit value.

## Artifacts

- Raw stream: `.compozy/tasks/network-threads/reviews-002/peer-review/impl-review-result-round1.json`
- Extracted output: `.compozy/tasks/network-threads/reviews-002/peer-review/impl-review-output-round1.md`
- Findings JSON: `.compozy/tasks/network-threads/reviews-002/peer-review/impl-review-findings-round1.json`
- Prompt: `.compozy/tasks/network-threads/reviews-002/peer-review/impl-review-prompt-round1.md`
- Patch: `.compozy/tasks/network-threads/reviews-002/peer-review/impl-review-diff-round1.patch`
