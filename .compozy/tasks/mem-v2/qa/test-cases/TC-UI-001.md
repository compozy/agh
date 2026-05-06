# TC-UI-001: Web Knowledge Surface Uses Memory v2 Contract Truthfully

**Priority:** P0
**Type:** UI
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Objective

Verify the Knowledge page uses generated Memory v2 selectors, server-backed search, controller-backed mutations, decision history, and truthful states without client-side filtering or legacy scope derivation.

## Preconditions

- [ ] Web dev server points at isolated daemon via `AGH_WEB_API_PROXY_TARGET`.
- [ ] Scenario daemon has global, workspace, agent-workspace, and agent-global memories.
- [ ] MSW/unit tests are available for focused regression.

## Test Steps

1. **Run focused web tests**
   - Input: `cd web && bunx vitest run src/routes/_app/-knowledge.test.tsx src/hooks/routes/use-knowledge-page.test.tsx src/systems/knowledge`
   - **Expected:** Route, hook, adapter, formatter, component, and story/fixture tests pass.

2. **Open Knowledge page**
   - Input: browser route for `/knowledge`.
   - **Expected:** Scope tabs and agent tier/name controls reflect `KnowledgeSelector` behavior.

3. **Search through server recall**
   - Input: search for scenario sentinel.
   - **Expected:** Network call uses `POST /api/memory/search`; UI does not fake results through client-only filtering.

4. **Edit/delete with controller feedback**
   - Input: edit then delete a memory.
   - **Expected:** UI shows controller decision context, preserves failure marker until selection/scope/search changes, and invalidates queries after mutation.

5. **Decision history panel**
   - Input: open decision history for the edited entry.
   - **Expected:** Redaction-safe decision metadata appears; raw replay content and raw LLM trace are absent.

6. **Negative copy/control check**
   - Input: inspect page text and controls.
   - **Expected:** No `read`, `consolidate`, filename-prefix scope derivation, speculative dashboard, or unsupported promote/replay controls.

## Evidence To Capture

- Focused web test log.
- Browser screenshot/DOM snapshot.
- Network request log showing `POST /api/memory/search`.
- Mutation response payloads.

