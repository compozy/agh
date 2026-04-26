## TC-AUTO-016: Runtime Autonomy Docs And CLI Reference Consistency

**Priority:** P1 (High)
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 40 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify `packages/site` runtime autonomy docs and generated CLI references match implemented MVP
behavior, preserve manual control, document token/channel boundaries, and avoid promising post-MVP
features.

### Traceability

- Task: task_16, Runtime Autonomy Docs And CLI References.
- TechSpec: Web and Docs Tests, Impact Analysis `packages/site` runtime docs, MVP boundary.
- ADR: ADR-002, ADR-003, ADR-005, ADR-006, ADR-009, ADR-010, ADR-011, ADR-012.
- Resource lesson: Hermes README/docs references favor precise operator-facing runtime docs over marketing claims.
- Surfaces: `packages/site/content/runtime/core/autonomy`, CLI reference pages, config docs, hook event catalog, runtime navigation.

### Preconditions

- Site dependencies installed.
- Current CLI reference pages generated from Cobra command metadata where applicable.
- Docs test suite includes autonomy coverage from task_16.

### Test Steps

1. Inspect autonomy overview, coordinator, task runs/leases, coordination channels, and safe spawn docs.
   - **Expected:** Docs state task creation does not enqueue work, publish/start/approval is the execution boundary, channels bind at run enqueue, and channels are conversation only.

2. Inspect CLI references for `agh me`, `agh ch`, `agh task next|heartbeat|complete|fail|release`, and `agh spawn`.
   - **Expected:** Commands and flags match implemented CLI; examples use implemented flags only.

3. Inspect config and hook docs.
   - **Expected:** `[autonomy.coordinator]` defaults/precedence and `coordinator.*`, `task.run.*`, `spawn.*` hook families match runtime behavior.

4. Search docs for post-MVP promises.
   - **Expected:** Cross-daemon swarm, built-in MCP mirrors, broad memory extraction, dashboards, and eval/replay are either absent or explicitly out of scope.

5. Run site verification gates.
   - **Expected:** `source:generate`, typecheck, tests, and build pass.

### Evidence To Capture

- `qa/logs/TC-AUTO-016/site-source-generate.log`
- `qa/logs/TC-AUTO-016/site-typecheck.log`
- `qa/logs/TC-AUTO-016/site-test.log`
- `qa/logs/TC-AUTO-016/site-build.log`
- `qa/logs/TC-AUTO-016/docs-scope-inspection.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Claim token examples | docs search | Raw token appears only as command input from claim response |
| Channel status wording | docs search | Never says channel status changes task status |
| Coordinator config | config docs | Defaults and precedence match task_01/task_14 |
| CLI generated page | regenerated docs | No stale manual tail after generation |

### Related Test Cases

- TC-AUTO-005: CLI command behavior.
- TC-AUTO-018: Post-MVP boundary.
