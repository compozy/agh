## TC-INT-003: OpenAPI + Generated TS Codegen Stay In Sync With `category_path`

**Priority:** P0
**Type:** Integration
**Module:** `make codegen`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Confirm that the OpenAPI schema and the TypeScript types generated for the web client both expose `category_path?: string[]` on every agent payload (`AgentResponse.agents`, single-agent endpoint, workspace agents, bundle activation agents) and that `make codegen-check` blocks any drift.

---

### Preconditions

- [ ] Branch HEAD with the implementation merged.
- [ ] `make codegen` and `make codegen-check` work locally.

---

### Test Steps

1. **Inspect `openapi/agh.json` for `category_path`.**
   - Input: `jq -r '.components.schemas | to_entries[] | select(.value.properties.category_path) | .key' openapi/agh.json`
   - **Expected:** Output includes the agent payload schemas (`AgentPayload`, `BundleAgentPayload`, or their named forms in this repo). Each definition declares `category_path` as `{ "type": "array", "items": { "type": "string" } }` with `nullable` semantics consistent with `omitempty` (i.e., not in the `required` list).

2. **Inspect generated TypeScript.**
   - Input: `rg "category_path" web/src/generated/agh-openapi.d.ts`
   - **Expected:** Each agent-shaped interface has `category_path?: string[]`. No `category_path: string` (singular) or `categories` exists.

3. **Re-run codegen and confirm no drift.**
   - Input: `make codegen` then `git diff -- openapi/agh.json web/src/generated/agh-openapi.d.ts`
   - **Expected:** Empty diff (the implementation already shipped the regenerated artifacts). Then `make codegen-check` exits 0.

4. **Negative drift probe.**
   - Input: Manually edit `openapi/agh.json` to remove `category_path` from one agent schema. Run `make codegen-check`.
   - **Expected:** `make codegen-check` exits non-zero with a clear drift message. Restore the file before continuing.

---

### Behavioral Evidence

- Cross-surface: contract source ↔ generated TS agree.
- Disruption probe: drift detection actually fires.

---

### Audit Coverage

- C5: contract + web codegen surfaces.
- C8: schema vs generated TS.
- C11: drift disruption.
- C14: `make codegen-check`.

---

### Pass Criteria

- All four steps pass; the negative drift probe must end with the file restored to clean state.

---

### Failure Criteria

- `category_path` missing from any agent-shaped schema.
- Generated TS uses a different shape (`string`, alias, etc.).
- `make codegen-check` does not catch the simulated drift.
