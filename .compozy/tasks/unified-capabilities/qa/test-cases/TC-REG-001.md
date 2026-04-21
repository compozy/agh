## TC-REG-001: Protocol reference and example consistency

**Priority:** P1
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-20
**Last Updated:** 2026-04-20

---

### Objective

Verify that the public protocol reference in `packages/site/content/protocol/` teaches the unified capability model consistently: `kind:"capability"` is the only transferred artifact, the three discovery/transfer roles are clear, examples are coherent, and the obsolete `recipes` page is gone from steady-state navigation.

---

### Preconditions

- [ ] Protocol pages from task_07 and RFC/source docs from task_05 are present.
- [ ] The executor can inspect source files and run `make site-build`.

---

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Nav metadata | `packages/site/content/protocol/meta.json` | Confirms steady-state navigation |
| Protocol pages | `message-kinds.mdx`, `capability-discovery.mdx`, `examples.mdx`, related pages | Primary review set |
| Repo RFC | `docs/rfcs/003_agh-network-v0.md` | Source of truth for wording |

---

### Test Steps

1. Inspect `packages/site/content/protocol/meta.json` and the protocol section file set.
   - **Expected:** There is no first-class `recipes.mdx` page in the steady-state protocol navigation.

2. Review `message-kinds.mdx`, `capability-discovery.mdx`, and `examples.mdx`.
   - **Expected:** `kind:"capability"` is the only transferred artifact kind, the brief/rich/transfer roles are explicit, and the examples use capability terminology end to end.

3. Spot-check adjacent protocol pages such as `peer-discovery.mdx`, `interactions.mdx`, and `nats.mdx`.
   - **Expected:** Capability discovery, directed/broadcast transfer, and lifecycle wording remain consistent with the updated message-kind contract.

4. Cross-check the site wording against `docs/rfcs/003_agh-network-v0.md`.
   - **Expected:** The site protocol reference matches the rewritten RFC rather than inventing a second explanation.

5. Build the docs site.
   - **Expected:** `make site-build` succeeds and there are no broken nav or content references caused by the rewrite.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Historical references | Any retained `recipe` mention | Clearly historical/superseded, never normative |
| Discovery wording | Brief vs rich vs transfer | Clear and non-overlapping |
| Protocol examples | Broadcast and directed capability flows | Match RFC 003 semantics |
| Nav integrity | Protocol landing and ordering | No broken page references after page removal/rewrite |

---

### Traceability

- Tasks: `task_07`, `task_05`
- TechSpec: `System Architecture`, `Technical Considerations`
- ADRs: `ADR-001`, `ADR-003`
- Primary surfaces: `packages/site/content/protocol/*`, `docs/rfcs/003_agh-network-v0.md`

---

### Evidence to Capture

- Source review notes or screenshots showing nav and key page sections
- `make site-build` output
- Any mismatch recorded as a linked `BUG-*.md`

---

### Notes

- This case proves public protocol coherence, not just source-file existence. A passing build is necessary but not sufficient if the narrative reintroduces a split model.
