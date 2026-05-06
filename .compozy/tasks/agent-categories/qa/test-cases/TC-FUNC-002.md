## TC-FUNC-002: `EditAgentDefFile` Round-Trips `category_path` Across Unrelated Mutations

**Priority:** P0
**Type:** Functional
**Module:** `internal/config`
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Verify that `EditAgentDefFile` preserves `category_path` on disk when the caller mutates an unrelated field (`Skills.Disabled`, prompt body, MCP servers, etc.). Without this guarantee, every skill toggle would silently drop authored category intent.

---

### Preconditions

- [ ] Temp workspace with an AGENT.md whose frontmatter includes `category_path: ["Engineering", "Tools"]` AND at least one entry under `skills.disabled`.
- [ ] Go test harness writable.

---

### Test Steps

1. **Load the AGENT.md.**
   - Input: `ParseAgentDef(...)`.
   - **Expected:** Parsed `AgentDef.CategoryPath` equals `["Engineering", "Tools"]`.

2. **Mutate an unrelated field via `EditAgentDefFile`.**
   - Input: Toggle `Skills.Disabled` (add or remove a skill).
   - **Expected:** Returns no error.

3. **Re-read the file from disk.**
   - Input: `os.ReadFile(...)` then `ParseAgentDef(...)`.
   - **Expected:** Frontmatter on disk still contains the `category_path: [Engineering, Tools]` array verbatim. Reparsed `AgentDef.CategoryPath` equals `["Engineering", "Tools"]`. Order preserved.

4. **Mutate `category_path` itself via `EditAgentDefFile`.**
   - Input: Replace with `["Ops", "Reliability"]`.
   - **Expected:** On-disk YAML now reflects `["Ops", "Reliability"]`; reparsed value equals exactly that.

5. **Repeat step 2 after step 4.**
   - Input: Toggle `Skills.Disabled` again.
   - **Expected:** `category_path: ["Ops", "Reliability"]` still survives the next write.

---

### Behavioral Evidence

- Artifact reuse: edited AGENT.md is reused by the next parse, proving disk → daemon round-trip stays coherent.
- Disruption probe: unrelated mutation cannot drop `category_path`.

---

### Audit Coverage

- C8: disk vs in-memory truth.
- C10: same AGENT.md is used as both write target and read source.
- C11: round-trip disruption.
- C14: `go test ./internal/config -run "TestEditAgentDef.*CategoryPath"`.

---

### Pass Criteria

- All five steps pass.
- The post-write YAML is structurally equivalent to the input (no key reordering that breaks readability is mandated, but `category_path` keeps its array form and segment order).

---

### Failure Criteria

- `category_path` is dropped after any unrelated mutation.
- Segment order changes after a write/read cycle.
- Field gets re-encoded as a slash-string.
