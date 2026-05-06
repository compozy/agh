# TC-INT-005: Dream Runtime and Promotion Artifacts

**Priority:** P1
**Status:** Not Run

## Preconditions

- Dreaming is enabled for the target workspace.
- Candidate memories exist for promotion.

## Steps

1. Trigger a dream run for a workspace.
2. Observe status and retrieve the dream record.
3. Retry the dream run if needed.

**Expected:** Dream execution is scoped to the requested workspace_id, writes artifacts under _system/dreaming, and reports promotion counts without losing audit linkage.

## Required Evidence

- Trigger response.
- Dream status response.
- Dream detail response.
- Retry response if exercised.
- Artifact proof for _system/dreaming.
