## TC-FUNC-004: CLI Outputs Expose `category_path` Across human / TOON / JSON

**Priority:** P0
**Type:** Functional
**Module:** `internal/cli`
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Confirm that `agh agent list`, `agh agent info`, and `agh workspace info` (or the equivalent agent-table view) surface the agent category in all three output formats with the correct shape:

- `human`: a `Category` column / detail row rendered as `Marketing / Sales`; root-level agents render `-`.
- `toon`: a `category` key with the same space-delimited path string.
- `json`: `category_path` exposed verbatim as a JSON array.

---

### Preconditions

- [ ] Daemon is running against a workspace with at least one categorized agent (multi-segment) and one root-level agent.
- [ ] CLI binary built from branch HEAD.

---

### Test Steps

1. **`agh agent list` human output.**
   - Input: `agh agent list -o human`
   - **Expected:** Table includes a `Category` column. The categorized agent's row contains `Marketing / Sales` (single space-delimited). The root-level agent's row contains `-`.

2. **`agh agent list` TOON output.**
   - Input: `agh agent list -o toon`
   - **Expected:** Each agent record includes a `category` key. Categorized agent is `category: "Marketing / Sales"`. Root-level agent is omitted or `category: "-"` per the existing TOON conventions; either way it must NOT contain a slash-string of arbitrary segments.

3. **`agh agent list` JSON output.**
   - Input: `agh agent list -o json`
   - **Expected:** Each agent object includes `"category_path"` as an array. Categorized agent: `"category_path": ["Marketing", "Sales"]`. Root-level agent: field omitted (`omitempty`).

4. **`agh agent info <name>` human output.**
   - Input: `agh agent info <categorized-agent> -o human`
   - **Expected:** Detail block has a `Category` row equal to `Marketing / Sales`.

5. **`agh agent info <name>` JSON output.**
   - Input: `agh agent info <categorized-agent> -o json`
   - **Expected:** `category_path` is the array `["Marketing", "Sales"]`.

6. **`agh workspace info` agent table.**
   - Input: `agh workspace info -o human` (or equivalent workspace command surfacing agents).
   - **Expected:** Table includes the same `Category` column with `Marketing / Sales` for the categorized agent and `-` for the root-level agent. JSON output exposes `category_path` as an array.

---

### Behavioral Evidence

- Agent-manageable surface: `-o json` is the path agents use to discover categories without the web UI.
- Cross-format parity: same agent, three formats, same logical value.

---

### Audit Coverage

- C4: operator + agent observers.
- C5: CLI human/toon/json surfaces.
- C8: parity across the three formats.
- C14: `go test ./internal/cli -run "AgentList|AgentInfo|Workspace"` plus a manual run captured under `qa/`.

---

### Pass Criteria

- All six steps pass.
- No format invents a slash-only string in JSON, no format omits the field in human/toon for a categorized agent.

---

### Failure Criteria

- Any format renders a different value than the others for the same agent.
- JSON ever emits `category` (singular string) or `categories` (alias).
- Human/toon ever renders a literal `nil` or `[]` for root-level agents instead of an empty cell or `-`.
