## TC-INT-001: Startup section selection follows resolved harness policy

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-18
**Workstream:** Workstream 1 and Workstream 2
**Traceability:** `task_01.md`, `task_02.md`, ADR-001, ADR-002

---

### Objective

Validate that daemon-owned context resolution drives startup prompt section
selection deterministically for both non-channel and channel-bound sessions, and
that the explicit `agh-network` startup section appears only when policy says it
should.

---

### Preconditions

- [ ] Current harness branch builds and the daemon/runtime integration lane is available
- [ ] An isolated AGH home and workspace can be created for this scenario
- [ ] One non-channel session and one channel-bound session can be started through repo-supported runtime paths
- [ ] Observe/query surfaces can return `event_summaries` for the created sessions

---

### Test Steps

1. **Start a baseline non-channel session through the normal daemon/runtime path**
   - **Expected:** Session starts successfully and the startup prompt is assembled without any explicit network overlay content.

2. **Start a channel-bound session through the same path**
   - **Expected:** Session starts successfully and the startup prompt includes the explicit network section exactly once, in the deterministic startup position defined by the section registry.

3. **Query harness lifecycle summaries for both sessions**
   - **Expected:** Each session shows `harness.context_resolved` followed by `harness.section_selected`, with session-specific payloads reflecting the correct policy inputs and selected section names/order.

4. **Resume or recreate the channel-bound session using the repo-supported resume path**
   - **Expected:** The resumed path uses the same resolver/selector behavior, does not duplicate the network section, and does not fall back to an inline append path.

5. **Compare the non-channel and channel-bound startup artifacts**
   - **Expected:** Baseline sections remain stable across both sessions, while the network section is present only in the channel-bound session.

---

### Evidence to Capture

- Session ids for both sessions
- Startup prompt excerpts showing the selected sections
- Ordered `harness.context_resolved` and `harness.section_selected` summaries
- A note confirming whether the resume path preserved section selection without duplication

---

### Edge Cases & Variations

| Variation | Input / Condition | Expected Result |
| --- | --- | --- |
| Non-channel baseline | user session with no channel binding | no network startup section selected |
| Channel-bound startup | user session with channel binding | explicit network startup section selected exactly once |
| Resume path | resume existing channel-bound session | same section-selection policy, no duplicate overlay |
| Missing channel metadata | session expected to be local/non-network | network section omitted |
| Summary association timing | startup summary emitted before session row exists | summary still lands on the correct session after session creation |

---

### Related Test Cases

- `TC-INT-002`: Ordered prompt augmentation preserves stored input
- `TC-INT-007`: Harness observability and HTTP/UDS parity

---

### Notes

Suggested repo-supported runtime anchors:

- `internal/daemon/harness_context_integration_test.go`
- `internal/daemon/composed_assembler_test.go`
- `internal/daemon/harness_observability_test.go`
