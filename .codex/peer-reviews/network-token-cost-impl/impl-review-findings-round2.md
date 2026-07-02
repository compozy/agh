---
schema_version: 1
review_kind: implementation
round: 2
verdict: SHIP
reviewer_runtime: claude
reviewer_model: opus
generated_at: 2026-07-02T02:51:47Z
---

# Summary

All six round-1 blockers (B-001..B-006) are remediated at the code level and, critically, are now
owned by behavioral tests — channel fanout policy is enforced in routing with coordinator/all_members/
capability_match branches, designated fan-out N≥2 works through the real store, digest mode coalesces
with honest per-envelope cost allocation, capability no-match downgrades to digest fallback, mute
exempts addressed traffic, and the subscription/mute/digest/activation machinery has real assertions.
Every round-1 risk except R-003 (partial fan-out orphans, now live because B-002 was fixed) is resolved,
generated artifacts co-ship correctly across openapi/web/sdk, CLI/HTTP/UDS/native parity is complete,
and no new blocker was found — the residual items are non-blocking risks and nits.

# Blockers

None.

# Risks

## R-201 — Partial fan-out failure leaves orphaned sibling runs (round-1 R-003, now live)

- File: internal/api/core/tasks.go
- Line: 1213
- Issue: `enqueueFanOutTaskRuns` (tasks.go:1213-1240) loops `manager.EnqueueRun` once per designation and returns on the first error, leaving already-enqueued siblings persisted as `queued` runs; the handler then returns 500 and `PutTaskDesignationRollup` is never written for that group (tasks.go:1185-1196). This was round-1 R-003, previously moot because B-002 made fan-out non-functional; now that B-002 is fixed and N≥2 works, the partial-failure path is reachable. The most likely trigger is a failure on designation ≥2 (transient DB error, per-designation metadata issue) after designation 1 already created a run. A client retry generates a fresh `groupID` (`store.NewID("tdg")`, tasks.go:1178) with per-index idempotency keys, so the orphan from the failed attempt is stranded under the old group while a new group is created. The observer self-heals partially (`recomputeAndSendDesignationRollup` recomputes from live runs), but the stray run still executes and posts status-back under an abandoned group.
- Suggested fix: Enqueue the designation group atomically (single tx spanning all `ReserveQueuedRun` calls), or on partial failure roll back the already-enqueued siblings; alternatively define and document partial-fan-out semantics and return the partial set + group id so callers can reconcile. Add a test asserting no orphan `queued` rows remain after a mid-group enqueue failure.

## R-202 — Rollup completion status-back can double-send under concurrent sibling termination

- File: internal/daemon/network_task_status_observer.go
- Line: 245
- Issue: `recomputeAndSendDesignationRollup` gates the completion message on a transition by reading the previously-persisted rollup via `previousDesignationRollupComplete` (observer.go:270-292), then persisting, then sending only when `!wasComplete && rollup.Complete`. The read-compute-persist-check-send sequence is not atomic. Sibling runs can execute in different sessions and terminate concurrently, firing two `run_completed` events into two observer invocations. If both read `wasComplete=false` before either persists, both compute `Complete=true` and both send, producing a duplicate `task_fanout_rollup` status-back into the thread. This is cosmetic noise (not data corruption) and low-probability (requires two siblings terminal within the observer window), but the dedup is best-effort rather than race-free.
- Suggested fix: Make the completion decision atomic — e.g. persist the rollup and detect the incomplete→complete transition inside a single `BEGIN IMMEDIATE` tx that returns whether this call was the one that flipped `Complete`, and send only on that signal; or serialize designation-group rollup handling per group id.

## R-203 — Positive-path activation and keyword-filter matches are untested

- File: internal/network/router_test.go
- Line: 436
- Issue: B-006 is substantially closed (11 of 13 delivery invariants now have `t.Run("Should …")` behavioral tests with real assertions), but two positive paths remain uncovered: (a) `capability_match` WITH a positive score — no test asserts that a matching peer is activated as `full` injection (only the no-match→digest-fallback path at router_test.go:436 is tested); (b) keyword filter WITH a match — only the miss case is tested (manager_test.go:1914 asserts digest fallback when the filter misses), so the "filter matches → deliver full" branch of `subscriptionKeywordFiltersMatch` (manager.go:1271-1286) is unexercised. Both are the "happy path" of features whose failure modes (over- or under-delivery) directly affect the round's token-cost goal.
- Suggested fix: Add two subtests to the canonical suites — one asserting a capability-matched peer receives `full` (Mode empty/full) when its catalog overlaps the intent terms, and one asserting a full-mode subscription with a matching keyword filter delivers `full`.

# Nits

## N-201 — Every digest delivery incurs the full flush-interval latency even for a single envelope

- File: internal/network/delivery.go
- Line: 688
- Issue: `collectDigestBatch` blocks the per-session delivery worker for the entire `digestFlushInterval` (default 250ms) whenever the head item is digest-mode and `digestMaxEnvelopes > 1`, even when no additional envelopes are waiting — adding a fixed 250ms latency to every digest delivery, not only coalesced ones.
- Suggested fix: Skip the flush wait when the queue holds no further digest-eligible items for the peer, or short-circuit the timer once the queue drains.

## N-202 — `task_designation_rollups` reader is used only by the dedup path

- File: internal/store/globaldb/global_db_network_preferences.go
- Line: 344
- Issue: `ListTaskDesignationRollups` now has exactly one non-test caller — the observer's `previousDesignationRollupComplete` dedup read — so the persisted rollup summary is never surfaced to an API/CLI/UDS/thread projection for agents or operators to read back the fan-out result.
- Suggested fix: Confirm the write-mostly rollup is intentional for this round, or wire a read surface (task detail / thread projection) so the aggregated designation result is observable.

# Evidence

Read in full (first-hand): the round-1 findings, the megaround packet
(`.compozy/tasks/network-token-optimization/codex-megaround-packet.txt`), the plan
(`.codex/plans/20260701T200023-0300*…megaround.md`), root `CLAUDE.md` and `internal/CLAUDE.md`;
`internal/network/router.go` (full), `internal/network/manager.go` (1-1649, incl.
`applyDeliverySubscriptions`/`deliverySubscriptionMode`), `internal/network/delivery.go` (1-1606,
incl. `collectDigestBatch`/`deliveredPromptCostAllocations`/`cappedStructuredDeliveryBody`),
`internal/store/globaldb/global_db_task_aux.go` (`ReserveQueuedRun`,
`findOpenRunIDForQueuedRunReservation`), `internal/daemon/network_response_register_prompt.go` (full),
`internal/daemon/truncate_utf8.go`, `internal/api/core/network_conversations.go` (promote handler),
`internal/api/core/tasks.go` (fan-out handler + enqueue), `internal/cli/network.go` (channel update).

Blocker verification (first-hand):

- B-001 FIXED — `selectEmptyThreadPeers` (router.go:1281-1317) branches on
  `NetworkFanoutPolicyAllMembers`, `NetworkFanoutPolicyCoordinator` (full-inject coordinator, digest
  fallback when missing), and default `capability_match`; policy resolved via
  `channelFanoutPolicy` (router.go:1425-1446). Tests: router_test.go:563 (coordinator), :637
  (all_members), :504 (coordinator unavailable → digest).
- B-002 FIXED — `findOpenRunIDForQueuedRunReservation` (global_db_task_aux.go:1477-1509) excludes
  same-group runs via `(? = '' OR COALESCE(designation_group_id,'') <> ?)`, so N≥2 siblings coexist
  as `queued` while non-designated/other-group open runs still block. Store test
  `TestGlobalDBReserveQueuedRunAllowsDesignatedSiblingRuns` (global_db_task_test.go:1234) proves N≥2
  through the real store; API test tasks_test.go:2050 asserts 201 + two runs sharing group id; the
  in-memory stub now replicates the designation-aware guard (manager_test.go:1305).
- B-003 FIXED — `collectDigestBatch` (delivery.go:688-706) coalesces up to `digestMaxEnvelopes` after
  `digestFlushInterval`; `formatNetworkDigestBatchMessage` (delivery.go:1471-1513) renders one prompt;
  `deliveredPromptCostAllocations` (delivery.go:1213-1254) splits cost by canonical body weight,
  last-item-gets-remainder (total preserved). Test delivery_test.go:867 asserts batch wrapper +
  `count="2"` + allocated bytes/tokens sum to the rendered message cost.
- B-004 FIXED — capability no-match returns `digestPeers` via `threadDigestFallbackPeers`
  (router.go:1307-1316), Mode=digest not full. Test router_test.go:436.
- B-005 FIXED — `deliverySubscriptionMode` exempts addressed traffic via `deliveryAddressedToPeer`
  (manager.go:1194). Tests manager_test.go:1967 (muted peer still gets addressed msg), :1946
  (muted peer still gets mentioned msg).
- B-006 SUBSTANTIALLY FIXED — 11/13 delivery invariants tested with `t.Run("Should …")` + `t.Parallel`;
  no test asserts the old broken behavior. Gaps: positive capability_match match and positive
  keyword-filter match (see R-203).

Round-1 risk/nit status (verified):

- R-001 FIXED (structured body cap wired: `withDeliveryStructuredBodyMaxBytes` manager.go:368;
  `cappedStructuredDeliveryBody` delivery.go:1425; test delivery_test.go:271).
- R-002 FIXED (guidance store read now outside the coordinator mutex: delivery.go:541-563).
- R-003 → carried forward as R-201 (now live).
- R-004 FIXED (per-sibling posts suppressed for designated runs; completion dedup via previous rollup
  state: observer.go:150-154, 245-266).
- R-005 FIXED (handler status+body tests: "Should update network channel policy" network_test.go:1424,
  "Should upsert and list network subscriptions…" :1457, `TestBaseHandlersPromoteNetworkThreadTask`
  :1596 with origin assertions, fan-out tasks_test.go:2050).
- R-006 FIXED (subscription upsert now ensures the channel row first:
  `ensureNativeNetworkSubscriptionChannel` native_tools.go, `networkChannelMetadataForUpdate`
  network_details.go — no opaque FK error).
- N-001 FIXED (`defaultRouterActivationTopK = 3`, router.go:122).
- N-002 FIXED (`truncateUTF8Bytes`, truncate_utf8.go; used at native_tools.go:3056 and
  observer truncateStatusText).
- N-004 FIXED (promote handler rolls back the created task via `DeleteTask` when origin write fails,
  network_conversations.go:149-156).
- N-005 → downgraded to N-202 (reader now consumed by dedup path only).
- N-006 FIXED (bare `PromptSection` returns `"", nil`, network_response_register_prompt.go:31-36).

Migrations v44-v47 (subagent, cross-checked against global_db.go registry): appended at the tail with
distinct checksums, no reorder/rename of v1-v43, wrapped in the framework transaction; workspace_id
present on `network_subscriptions` and `network_task_thread_origins`; guidance-state/rollups correctly
session-/task-scoped; migration-tail assertion + fresh-DB + reopen tests updated for v47.

Generated-artifact co-ship (first-hand, corrected a subagent false positive): `openapi/agh.json`,
`web/src/generated/agh-openapi.d.ts`, and `sdk/typescript/src/generated/contracts.ts` are all modified
in the working tree. `contracts.ts` is the extension **Host API** contract (header: "generated by go run
./cmd/agh-codegen sdk-contracts"), a deliberate subset — its 8-line diff adds `fanout_policy`,
`coordinator_peer_id`, `mentions` (×3 payloads), `designation_group_id` (×2) and `RunDesignationSummary`;
the REST subscription/promote endpoints are correctly absent because they are not Host API methods. No
codegen drift. Route path templates in openapi.json match the gin registrations for the four new
endpoints.

Parity (subagent, corrected a subagent false positive): channel-update, subscription list/set/delete,
thread promote, and task fan-out each have HTTP + UDS + core handler + native tool + CLI coverage. The
CLI channel-update command DOES exist (`agh network channels update` with `--fanout-policy` /
`--coordinator-peer-id`, network.go:310, using `Changed()` flag-presence detection); the native-tool
catalog fixture includes all eight new tool ids.

Limitations: I did not run `make verify`, `make codegen-check`, or the test suites (read-only review of
the working tree under commit `669f6b503 savepoint`). Manager/delivery were read to line 1649/1606 of
2066/1988; the relevant subscription/digest/cost functions were fully in view. Migration, test-coverage,
status-observer, subscription-surface, and codegen findings were gathered via read-only Explore
subagents and independently re-verified for every claim that could affect the verdict (the codegen-drift
and missing-CLI claims from subagents were checked first-hand and found to be false positives).

# Deferred Or Follow-Up

- R-201: make designated fan-out atomic (or define partial-failure semantics) before autonomous fan-out
  runs at scale; add the orphan-run test.
- R-202: close the rollup-completion double-send race with a transactional transition detector.
- R-203: add the two positive-path delivery tests (capability match, keyword-filter match).
- N-201/N-202: tune the digest flush-wait for single envelopes; decide whether the rollup needs a read
  surface.
- Pre-existing (out of this diff's fault, carried from round 1): `networkChannelEntry` lookup at
  global_db_task_claim.go filters by `channel` only (not workspace_id) — close when next touched.
