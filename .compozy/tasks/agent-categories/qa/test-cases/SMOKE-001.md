## SMOKE-001: Agent Categories - Build Readiness

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Confirm the branch can build and that generated artifacts agree before any release-grade case runs. This is entry criteria only and does not satisfy any release-grade behavioral minimum on its own.

---

### Preconditions

- [ ] Branch `agent-categories` checked out at the head referenced by Opus round-4 verdict (SHIP / 0 blockers).
- [ ] Bun and Go toolchains installed.
- [ ] Bootstrap manifest exists (or QA reuse manifest is healthy) under `.compozy/tasks/agent-categories/qa/bootstrap-manifest.json`.

---

### Test Steps

1. **Run the canonical guardrail.**
   - Input: `make verify`
   - **Expected:** Exit 0; `codegen-check`, `bun-lint`, `bun-typecheck`, `bun-test`, `web-build`, `fmt`, `lint`, `test`, `build`, `boundaries` all green.

2. **Confirm OpenAPI / TS codegen are in sync.**
   - Input: `make codegen` then `git status -- openapi/agh.json web/src/generated/agh-openapi.d.ts` then `make codegen-check`
   - **Expected:** No drift after `make codegen`; `make codegen-check` exits 0.

3. **Confirm bundled skill `SKILL.md` references the canonical field name.**
   - Input: `rg "category_path" internal/skills/bundled/skills/agh-agent-setup/SKILL.md`
   - **Expected:** At least one match for `category_path`; no match for `categories:` alias.

4. **Confirm packages/ui re-exports the tree primitives.**
   - Input: `rg "Tree, TreeItem, TreeItemLabel, TreeDragLine" packages/ui/src/index.ts`
   - **Expected:** Exactly one match.

---

### Audit Coverage

- This case satisfies no release-grade behavioral minimum by itself. It only proves the lab can run release-grade cases.
- Failure here BLOCKS execution of TC-FUNC-*, TC-INT-*, TC-UI-*, TC-SCEN-001, and TC-REG-*.

---

### Pass Criteria

- All four steps pass.
- A failure produces a `BUG-*` report and blocks the rest of the suite.

---

### Failure Criteria

- Any guardrail fails or codegen drifts after a clean run.
- Bundled skill references `categories:` alias.
