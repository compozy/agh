## TC-REG-002: `internal/workspace.cloneAgentDefs` Preserves Skills AND `CategoryPath`

**Priority:** P1
**Type:** Regression
**Module:** `internal/workspace/clone.go` + `internal/config/agent_clone.go`
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

The hand-rolled `cloneAgentDefs` body in `internal/workspace/clone.go` previously dropped `Skills` silently. The fix replaced the body with a delegation to `aghconfig.CloneAgentDef`. This case locks in both the regression fix (`Skills` survives) AND the new field (`CategoryPath` survives) so that any future field added to `AgentDef` is force-multiplied through the single clone authority.

---

### Context

Recent changes that may affect this case:

- `cloneAgentDefs` body deleted; now calls `aghconfig.CloneAgentDef`.
- `aghconfig.CloneAgentDef` updated with defensive copy for `CategoryPath`.

---

### Test Steps

1. **`Skills` regression fix.**
   - Input: Source `[]AgentDef` with `Skills.Disabled = ["a", "b"]` on at least one agent. Clone via `cloneAgentDefs`.
   - **Expected:** Cloned agents preserve `Skills.Disabled` exactly. Mutating the source `Skills.Disabled` slice after clone does NOT affect the clone.

2. **`CategoryPath` preservation.**
   - Input: Source agent with `CategoryPath = ["Engineering", "Tools"]`. Clone via `cloneAgentDefs`.
   - **Expected:** Clone has identical `CategoryPath`. Mutating the source slice does NOT affect the clone.

3. **Empty source returns nil.**
   - Input: `cloneAgentDefs(nil)` and `cloneAgentDefs([]AgentDef{})`.
   - **Expected:** Both return nil. No allocation of empty slices.

4. **Clone of a clone is stable.**
   - Input: `clone1 := cloneAgentDefs(src); clone2 := cloneAgentDefs(clone1)`.
   - **Expected:** `clone2` equals `src` for `Skills`, `CategoryPath`, `MCPServers`, `Hooks`, etc.

---

### Critical Path Tests

- [ ] No future field can drift between source and clone because `cloneAgentDefs` only delegates.

---

### Audit Coverage

- C4: workspace clone actor.
- C5: workspace boot path.
- C8: source vs clone parity.
- C11: source-mutation disruption.
- C14: `go test ./internal/workspace -run "TestCloneAgentDefs"`.

---

### Pass Criteria

- All four steps pass.
- The hand-rolled body in `internal/workspace/clone.go` is verified deleted (`rg "AgentDef{" internal/workspace/clone.go` returns no struct-literal matches outside of the delegation).

---

### Failure Criteria

- Any field is dropped or shared by reference between source and clone.
- The hand-rolled clone body returns.
