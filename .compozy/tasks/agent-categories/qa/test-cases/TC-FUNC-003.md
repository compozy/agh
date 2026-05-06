## TC-FUNC-003: `category_path` Validation Rejects Unsafe Segments

**Priority:** P0
**Type:** Functional
**Module:** `internal/config`
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Verify that `validateAgentCategoryPath` rejects every segment shape the file system or UI cannot safely render and that the error messages are stable enough for the daemon diagnostic surface (`AgentDiagnosticPayload`) to point at the offending index.

---

### Preconditions

- [ ] Go toolchain available.
- [ ] Test harness can construct in-memory AGENT.md text with arbitrary `category_path`.

---

### Test Steps

1. **Reject blank segment.**
   - Input: `category_path: ["Marketing", ""]`.
   - **Expected:** `AgentDef.Validate()` returns an error containing `agent.category_path[1]` and `blank segment`.

2. **Reject whitespace-only segment.**
   - Input: `category_path: ["   "]`.
   - **Expected:** Validation fails with `blank segment` after normalization trims to "".

3. **Reject `.` segment.**
   - Input: `category_path: ["."]`.
   - **Expected:** Validation fails with a message naming the invalid segment.

4. **Reject `..` segment.**
   - Input: `category_path: [".."]`.
   - **Expected:** Validation fails with the invalid-segment message.

5. **Reject forward slash inside a segment.**
   - Input: `category_path: ["Marketing/Sales"]`.
   - **Expected:** Validation fails with `must not contain '/' or '\\'`.

6. **Reject backslash inside a segment.**
   - Input: `category_path: ["Marketing\\Sales"]`.
   - **Expected:** Validation fails with `must not contain '/' or '\\'`.

---

### Behavioral Evidence

- Operator journey: malformed AGENT.md surfaces as an `agent_diagnostic` instead of a permissive parse with a corrupt category.
- Disruption probe: each negative case confirms there is no fallback (no slash split, no synthetic folder).

---

### Audit Coverage

- C4: parser + diagnostic actors.
- C11: validation disruption.
- C14: `go test ./internal/config -run "TestParseAgentDef_ShouldReject"` (or equivalent).

---

### Pass Criteria

- All six negatives fail with stable, indexed error messages.
- No invalid input parses successfully or is silently corrected.

---

### Failure Criteria

- Any malformed segment is accepted.
- Error message lacks the index or the violated rule, breaking diagnostic UX.
