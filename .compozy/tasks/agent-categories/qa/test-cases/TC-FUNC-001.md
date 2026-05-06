## TC-FUNC-001: AGENT.md `category_path` Parses, Normalizes, and Survives Validation

**Priority:** P0
**Type:** Functional
**Module:** `internal/config`
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Verify that `ParseAgentDef` populates `AgentDef.CategoryPath` from frontmatter exactly as authored, that whitespace is trimmed without altering casing or order, and that valid paths pass `AgentDef.Validate()`. Empty / missing values land as `nil` (no synthetic folder, no slash-string).

---

### Preconditions

- [ ] Go toolchain available; `make test` works against `./internal/config`.
- [ ] AGENT.md fixture at `internal/config/testdata/...` (or in-test buffer) exposing the relevant frontmatter shapes.

---

### Test Steps

1. **Parse a multi-segment categorized agent.**
   - Input: AGENT.md with `category_path: ["Marketing", "Sales"]`.
   - **Expected:** `AgentDef.CategoryPath` equals `["Marketing", "Sales"]` exactly. Casing preserved, order preserved, no slash-joining.

2. **Parse an agent with no `category_path` key.**
   - Input: AGENT.md without `category_path`.
   - **Expected:** `AgentDef.CategoryPath` is `nil` (root-level). No synthetic `Uncategorized` placeholder injected.

3. **Parse an agent with `category_path: []`.**
   - Input: AGENT.md with empty array.
   - **Expected:** `AgentDef.CategoryPath` is `nil` (normalization collapses empty to nil).

4. **Parse an agent with whitespace-padded segments.**
   - Input: `category_path: ["  Marketing  ", "Sales"]`.
   - **Expected:** `AgentDef.CategoryPath` equals `["Marketing", "Sales"]`. Trim only — no lowercase, no dedupe, no reorder.

5. **Round-trip via `CloneAgentDef`.**
   - Input: Mutate the source slice after `CloneAgentDef`.
   - **Expected:** Clone is unaffected (defensive copy). Source mutation does not leak into the clone.

---

### Behavioral Evidence

- Operator journey: agent author edits AGENT.md and the runtime sees the categorized agent intact.
- Cross-surface: `AgentDef.CategoryPath` value matches AGENT.md text byte-for-byte (after trim).
- Disruption probe: re-parse the same file twice; the second parse must equal the first.

---

### Audit Coverage

- C4: parser actor.
- C8: disk truth → in-memory `AgentDef` parity.
- C11: trim disruption.
- C14: `go test ./internal/config -run "ParseAgentDef.*CategoryPath"`.

---

### Pass Criteria

- All five steps pass with exact equality on `[]string`.
- `validate()` returns nil for every valid input.

---

### Failure Criteria

- Casing or order is mutated.
- Trim is skipped or extends to lowercasing.
- Empty / missing input becomes a non-nil slice or a synthetic folder.
