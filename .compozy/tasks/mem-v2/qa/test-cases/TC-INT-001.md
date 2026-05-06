# TC-INT-001: Controller WAL, Replay, Revert, And Atomic Mutation

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Objective

Verify the single-write-path invariant: every curated mutation persists a `memory_decisions` row with replay material before file/catalog mutation, applies atomically, emits `memory_events`, and can replay/revert deterministically.

## Preconditions

- [ ] Isolated workspace DB is available.
- [ ] Controller is configured for deterministic fresh-slot and update-slot decisions.
- [ ] Test fixture includes ADD, UPDATE, DELETE, NOOP, and REJECT candidates.

## Test Steps

1. **Run focused controller/store tests**
   - Input: `go test ./internal/memory/controller ./internal/memory -run "TestController|TestStore_Replay|TestStore_Atomic|TestStore_Revert" -count=1`
   - **Expected:** Tests pass and include ADD/UPDATE/DELETE/NOOP/REJECT coverage.

2. **Run race-sensitive write path tests**
   - Input: `go test -race ./internal/memory/controller ./internal/memory -count=1`
   - **Expected:** No race, deadlock, or SQLite busy leak.

3. **Exercise live ADD and UPDATE through public API**
   - Input: write a fresh entity slot, then write the same entity/attribute with different content.
   - **Expected:** First decision is ADD; second decision is UPDATE with `prior_content`, `post_content_hash`, `target_filename`, and idempotency key persisted.

4. **Simulate pending replay**
   - Input: create or use a fixture with `applied_at IS NULL`, restart daemon, then inspect file/catalog.
   - **Expected:** Replay reconstructs curated state exactly once and stamps/applies idempotently.

5. **Revert the update**
   - Input: `agh memory decisions revert <id> --dry-run`, then actual revert.
   - **Expected:** Dry-run previews the target; actual revert restores prior content and emits revert event.

6. **Negative path: rejected payload**
   - Input: candidate containing invisible Unicode or prompt-injection marker.
   - **Expected:** Decision is REJECT, content is not written, audit event is redaction-safe.

## Evidence To Capture

- Go test logs.
- Decision rows before/after replay.
- Curated file checksums.
- Revert response and resulting show/search output.
- Rejection response and event metadata.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|---|---|---|
| Duplicate idempotency key | Same candidate twice | Second apply is idempotent or rejected deterministically without duplicate file/catalog rows |
| Different post content | Same selector, changed content | Distinct idempotency key and UPDATE decision |
| Partial file write fault | injected write failure | Existing file remains intact |

