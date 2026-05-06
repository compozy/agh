---
provider: manual
pr:
round: 1
round_created_at: 2026-05-06T02:21:18Z
status: resolved
file: internal/memory/extractor/inbox.go
line: 322
severity: high
author: claude-code
provider_ref:
---

# Issue 004: Inbox multi-candidate file partial-apply breaks DLQ replay idempotency

## Review Comment

Safety invariant 9 in the TechSpec mandates that DLQ replay is deterministic and idempotent (`.compozy/tasks/mem-v2/_techspec.md` §Safety Invariants):

> "DLQ replay determinism. Failed extractions land in `_system/extractor/failures/<run_id>.json` containing the turn payload, prompt_version, model, coalesced_with ranges, and an `idempotency_key`. … Replay is idempotent — re-running produces the same decision events."

The extractor-inbox consumer cannot satisfy this invariant for any inbox file that contains more than one candidate. `InboxConsumer.consumeFile` (`internal/memory/extractor/inbox.go:306-342`) iterates candidates inside a single processing file and calls `c.sink.ProposeCandidate` per candidate:

```go
for _, candidate := range candidates {
    decision, err := c.sink.ProposeCandidate(ctx, candidate)
    if err != nil {
        result.Failed++
        failurePath, moveErr := c.moveToDLQ(processing, "controller", err)
        result.Failures = append(result.Failures, failurePath)
        c.recordFailure(ctx, candidate.Metadata["session_id"], failurePath, "controller", err)
        return result, errors.Join(...)
    }
    result.Proposed++
    result.Decisions = append(result.Decisions, decision)
}
```

`Store.ProposeCandidate` (`internal/memory/decision.go:143-185`) fully *applies* each candidate via `controller.New(s).Decide` followed by `s.ApplyDecision(ctx, decision)`. `ApplyDecision` (`internal/memory/decision.go:213-266`) writes a `memory_decisions` row first, then mutates the file, then marks `applied_at`. So when the sequence `[c1, c2, c3]` hits a failure on `c2`:

1. `c1` is fully applied — `memory_decisions` row inserted with its `idempotency_key`, file written, `applied_at` set.
2. `c2` fails. The whole `processing` file is moved to `_system/extractor/failures/<utc>-<base>.json` carrying all three candidates inside the wrapped `content` field (`inbox.go:377-403`).
3. The operator triggers `agh memory extractor replay --from-dlq` (or any retry that re-feeds the inbox file). The replay walks `[c1, c2, c3]` again. For `c1`, the controller produces the same `idempotency_key` (composed of `target_filename`, `op`, `post_content_hash`, `frontmatter_hash`, `prompt_version` — see `controller.go:475`).
4. `Store.catalog.insertDecision` (`decision.go:597-650`) executes a plain `INSERT INTO memory_decisions … VALUES …`. There is no `ON CONFLICT(idempotency_key) DO NOTHING` and no pre-check of the existing row, so the second insert fails with the SQLite UNIQUE constraint on `idempotency_key`.
5. `ApplyDecision` returns the wrapped error. The replay halts before reaching `c2` or `c3` — the very candidates the DLQ exists to retry.

Effectively the DLQ contract is "idempotent only when the failing file had exactly one candidate". Anything coalesced via the bounded queue (`runtime.go:259` `mergeRequests`) routinely produces multi-candidate files, so this is the common case in production.

Suggested fix:

- Change `insertDecision` to `INSERT INTO memory_decisions … ON CONFLICT(idempotency_key) DO NOTHING RETURNING id` and treat the no-rows result as "already applied — fall through to the existence check". The downstream `markDecisionApplied` already tolerates the no-op via the `applied_at IS NULL` filter; the only gap is the initial insert.
- Alternatively, scope the per-file processing as a single atomic decision-batch: defer all `ApplyDecision` calls until every candidate decoded successfully, then apply them in order, and on the first error roll back any persisted decision rows whose `applied_at` is still NULL.
- Add `TestExtractor_DLQReplayIsIdempotent` (already enumerated in §Test Plan §Extractor) that:
  - writes a JSONL inbox file with two candidates; first applies, second fails on a controlled error;
  - moves the file to DLQ;
  - replays the DLQ;
  - asserts the second candidate now applies cleanly and the first does not error or duplicate.
- Audit the `consumeFile` partial-apply story too — even without DLQ replay the partial-apply leaves disk and WAL state diverged from the inbox file's intent, which the spec calls out under "DLQ Replay determinism".

## Triage

- Decision: `VALID`
- Root cause: `Store.ApplyDecision` inserted every decision before mutation without checking the persisted idempotency key. A multi-candidate inbox file that partially applied before failing would replay the already-applied first candidate and halt on the unique `memory_decisions.idempotency_key` constraint before reaching the later failed candidates.
- Fix approach: make `ApplyDecision` load by idempotency key before insert, treat already-applied decisions as idempotent no-ops, still apply previously inserted pending decisions, and update tests so DLQ-style replay proves duplicate first decisions do not block the remaining candidates.

## Resolution

- Made `ApplyDecision` replay-safe by loading decisions by idempotency key before insert, returning already-applied decisions as no-op replays and applying previously persisted pending decisions.
- Added DLQ-style multi-candidate replay coverage proving an already-applied first candidate does not block a later retried candidate.
- Verification: `go test ./internal/memory ./internal/memory/extractor -count=1` passed; `go test -race ./internal/memory ./internal/memory/extractor -count=1` passed as part of the affected race run; `make verify` passed with Bun 334 files / 2150 tests, Go `DONE 8393 tests in 90.274s`, and boundaries OK.
