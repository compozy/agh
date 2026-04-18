## TC-FUNC-009: Task multi-agent live panel and fallback states

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Task Multi-Agent Live
**Route / Surface:** `/_app/tasks/$id` Agents panel
**Design Reference:** `docs/design/paper/tasks/AGH Tasks — Multi-Agent Live@2x.png`
**Execution Lane:** Manual + targeted browser regression

### Objective

Verify the task detail route can switch into the multi-agent live experience and present parent/descendant execution state without N+1 detail fetching or stitched session-first behavior.

### Preconditions

- [ ] A seeded task tree exists with at least one descendant or a deterministic no-descendant fallback state.
- [ ] The task detail route is reachable.

### Test Steps

1. Open a task detail route that supports the Agents panel.
   **Expected:** The live-mode or Agents entrypoint is visible from the detail flow.
2. Switch into the multi-agent live panel.
   **Expected:** Parent and descendant cards render with hierarchy cues and the shared detail shell remains stable.
3. Inspect active-run, latest-activity, and linked-session or run affordances.
   **Expected:** The live data reflects the tree read without requiring separate descendant detail fetches.
4. Exercise one live-navigation path or fallback condition.
   **Expected:** Navigation into run detail or session works when present; otherwise the fallback state is explicit and non-broken.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| No descendants | seeded tree has only the root task | the panel shows a clean no-descendants fallback |
| No active runs | descendant tree exists but nothing is running | live-state messaging stays useful without inventing activity |
| Stream disconnected | live refresh unavailable | a disconnected or stale-state fallback appears without collapsing the layout |

### Related Test Cases

- `TC-FUNC-007`
- `TC-FUNC-008`

### Notes

- This case is P1, but task_19 may use it to satisfy the "dashboard/inbox or live-state navigation" browser requirement once the core P0 Tasks flow is stable.
