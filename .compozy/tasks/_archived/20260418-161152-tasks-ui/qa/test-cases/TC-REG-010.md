## TC-REG-010: Settings route-presence preflight and blocker handling

**Priority:** P0
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Settings Preflight
**Route / Surface:** `web/src/routes/_app/settings*.tsx` presence check plus running-app entry validation
**Design Reference:** `docs/design/paper/settings/*.png`
**Execution Lane:** Manual + browser preflight

### Objective

Verify the execution branch actually contains a shipped Settings route family before `task_19` attempts to add or run Settings browser coverage.

### Preconditions

- [ ] The execution branch is checked out locally.
- [ ] The repo file tree is available.
- [ ] The daemon-served app shell can be launched if route files exist.

### Test Steps

1. Check whether `web/src/routes/_app/settings*.tsx` or an equivalent shipped Settings route family exists on the execution branch.
   **Expected:** The branch either exposes a Settings route family or clearly does not.
2. If route files exist, start the app and attempt to reach the Settings surface from the shipped UI entrypoint.
   **Expected:** The Settings shell is reachable and ready for downstream execution cases.
3. If route files do not exist or the UI entrypoint is not wired, record the result as a blocker in `.compozy/tasks/tasks-ui/qa/verification-report.md`.
   **Expected:** `TC-REG-011` through `TC-REG-014` are marked blocked rather than silently skipped.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| No route files | current branch state | Settings execution is blocked and explicitly reported |
| Route files exist but shell entry is broken | UI renders no reachable Settings entry | treated as a blocker, not as a skipped or optional path |
| Alternate route naming | equivalent generated or nested route exists | document the actual route family and continue with downstream Settings cases |

### Related Test Cases

- `TC-REG-011`
- `TC-REG-012`
- `TC-REG-013`
- `TC-REG-014`

### Notes

- On the planning branch for task_18, this preflight currently points to an absent Settings route family. Task_19 must re-run the preflight on its execution branch rather than assuming the result is unchanged.
