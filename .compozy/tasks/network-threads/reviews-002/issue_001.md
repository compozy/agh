---
provider: cy-impl-peer-review
pr: local-uncommitted
round: 2
round_created_at: 2026-05-05T18:30:17Z
status: resolved
file: web/src/systems/network/components/directs/direct-room.tsx
line: 86
severity: high
author: cy-codex-loop
provider_ref: B-001
---

# Issue 001: Implementation peer review blocker

## Review Comment

The missing-direct error copy still says `AGH could not load ${directId}. Chose an existing direct room from #${channel}.` while the structurally identical missing-thread copy at `thread-overlay.tsx:34` was just reworked to drop the `AGH` prefix in the round-001 remediation. The diff therefore ships two contradictory phrasings for the same UX state on adjacent surfaces.

## Rationale

Round-001 issue 001 explicitly classified the `AGH ...` wording as a `COPY.md` violation and called for the description to use the failed resource as the subject. The triage applies to `direct-room.tsx` because it uses the same `ConversationError` pattern and the same operator action. Shipping the fix on only one of the two matching surfaces leaves the copy hard cut incomplete.

## Suggested Fix

In `direct-room.tsx`, replace the description with `Could not load direct room ${directId}. Choose an existing direct room from #${channel}.`, update the corresponding direct-room test expectation if it asserts the description text, and rerun `make verify`.

## Triage

- Decision: `VALID`
- Notes: The direct-room missing-detail state uses the same `ConversationError` pattern and operator action as the thread missing-detail state fixed in round 001. `COPY.md` requires Web UI microcopy to tell the operator what is true and what action is available; using the product acronym as the subject adds implementation/product naming noise without clarifying the state.
- Root cause: the round-001 copy fix removed `AGH` only from the thread detail error path and missed the parallel direct-room error path.
- Fix approach: update the direct-room error description to use `Could not load direct room ${directId}. Choose an existing direct room from #${channel}.`, then add a test assertion for the description and run the full repository gate.
- Resolution: updated `web/src/systems/network/components/directs/direct-room.tsx` to render `Could not load direct room ${directId}. Choose an existing direct room from #${channel}.` and updated `web/src/systems/network/components/directs/direct-room.test.tsx` to assert the description text.
- Verification: `bunx vitest run web/src/systems/network/components/directs/direct-room.test.tsx` exited 0 with `1 passed` file and `5 passed` tests. `make verify 2>&1 | tee .compozy/tasks/network-threads/reviews-002/verify-after-fix.log` exited 0; Bun lint reported `Found 0 warnings and 0 errors`; Vitest reported `355 passed` files and `2223 passed` tests; Web build completed; Go lint reported `0 issues`; Go tests reported `DONE 8401 tests`; boundaries reported `OK: all package boundaries respected`.
