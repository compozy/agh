## TC-INT-002: Resource Codec / Daemon Sync Round-Trip `category_path`

**Priority:** P0
**Type:** Integration
**Module:** `internal/config` + `internal/resources` + `internal/daemon`
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Verify that `validateAgentResourceSpec` normalizes and re-validates `category_path`, that the resource codec round-trips the field through write → read, and that daemon resource sync surfaces validation errors via `errors.Is(err, resources.ErrValidation)` rather than swallowing them.

---

### Preconditions

- [ ] Daemon up against isolated `AGH_HOME`.
- [ ] Resource API reachable for agent resources.
- [ ] Test harness can post a spec via the resource codec.

---

### Test Steps

1. **Normalize whitespace via the resource codec.**
   - Input: Spec with `category_path: ["  Marketing  ", "Sales"]`.
   - **Expected:** Stored / re-read value is `["Marketing", "Sales"]`. Casing preserved.

2. **Reject invalid segments through the resource codec.**
   - Input: Spec with `category_path: ["Marketing/Sales"]` (slash inside segment).
   - **Expected:** Resource API returns an error such that `errors.Is(err, resources.ErrValidation)` is true. The stored state is unchanged.

3. **Round-trip through the resource store.**
   - Input: Write a categorized agent via the resource API, then read it back.
   - **Expected:** Read result equals the normalized write input (defensive copy verified by mutating the source after write — the stored value is untouched).

4. **Daemon resource sync.**
   - Input: Start the daemon against a workspace whose agent resource has `category_path: ["A", "B"]`. Inspect the sync log / status.
   - **Expected:** No diagnostic; the agent appears in `/api/agents` with `category_path: ["A", "B"]`.

5. **Daemon resource sync with invalid segment.**
   - Input: Pre-populate a resource spec with `category_path: ["", "B"]` (e.g., from disk). Restart the daemon.
   - **Expected:** Resource sync emits a structured error / diagnostic; the agent is exposed via `agent_diagnostics` with `category_path: nil` and a clear `agent.category_path[0]: blank segment` message.

---

### Behavioral Evidence

- Cross-surface: resource codec output equals subsequent reads.
- Disruption probe: invalid segments raise validation rather than silent corruption.

---

### Audit Coverage

- C5: resource API surface.
- C8: codec write vs read.
- C11: validation disruption.
- C14: `go test ./internal/config -run "AgentResource"` plus daemon integration coverage.

---

### Pass Criteria

- All five steps pass.
- Validation errors are wrapped to satisfy `errors.Is(err, resources.ErrValidation)`.

---

### Failure Criteria

- Invalid segments are stored.
- Defensive copy gap allows source mutation to corrupt stored state.
- Daemon swallows validation errors instead of surfacing them.
