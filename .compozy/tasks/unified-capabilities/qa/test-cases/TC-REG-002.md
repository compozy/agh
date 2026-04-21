## TC-REG-002: Runtime docs and repo-guide consistency

**Priority:** P1
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-20
**Last Updated:** 2026-04-20

---

### Objective

Verify that the runtime-facing documentation in `packages/site/content/runtime/core/agents/` stays aligned with `docs/agents/capabilities.md`: current layouts remain supported, `version` is optional, `digest` is runtime-computed, `requirements` reference `capability.id`, and no page teaches `recipe` as a separate authored/runtime concept.

---

### Preconditions

- [ ] Runtime site docs from task_08 and repo docs from task_05 are present.
- [ ] The executor can inspect source files and run `make site-build`.

---

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Runtime pages | `capabilities.mdx`, `definitions.mdx`, `meta.json` | Primary review set |
| Repo guide | `docs/agents/capabilities.md` | Runtime documentation source of truth |
| Related runtime pages | `agent-md.mdx`, overview pages if needed | Used to confirm no drift in surrounding narrative |

---

### Test Steps

1. Review `packages/site/content/runtime/core/agents/capabilities.mdx`.
   - **Expected:** The page documents supported layouts, required/optional fields, runtime-computed `digest`, and `requirements` semantics consistently with the unified model.

2. Review `packages/site/content/runtime/core/agents/definitions.mdx` and related metadata.
   - **Expected:** Agent-definition docs describe the capability sidecar as the unified discovery/transfer artifact and do not imply a separate recipe concept.

3. Cross-check the runtime site wording against `docs/agents/capabilities.md`.
   - **Expected:** The site and repo guide agree on no-catalog behavior, typed API guidance, discovery roles, and transfer semantics.

4. Spot-check surrounding runtime pages only where they reference capability behavior.
   - **Expected:** Capability-vs-skill explanations stay clear and no stale split-model wording remains in overview/configuration pages.

5. Build the docs site.
   - **Expected:** `make site-build` succeeds and the runtime section remains navigable after the rewrite.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| No-catalog behavior | Agent without capability files | Docs say load succeeds and discovery is empty |
| Remote requirements | `requirements` references missing local ID | Docs allow remote references, not local hard failure |
| Typed API guidance | API-visible payload explanation | Points readers to typed fields, not raw `ext` blobs |
| Runtime vs protocol boundary | Capability authoring vs wire rules | Runtime docs defer deep wire details to protocol docs |

---

### Traceability

- Tasks: `task_08`, `task_05`
- TechSpec: `System Architecture`, `Data Models`, `Technical Considerations`
- ADRs: `ADR-001`, `ADR-002`
- Primary surfaces: `packages/site/content/runtime/core/agents/*`, `docs/agents/capabilities.md`

---

### Evidence to Capture

- Source review notes or screenshots of key runtime-doc sections
- `make site-build` output
- Any mismatch recorded as a linked `BUG-*.md`

---

### Notes

- This case is the operator-facing documentation gate. It prevents the site runtime docs from drifting away from the repo guide after the protocol rewrite.
