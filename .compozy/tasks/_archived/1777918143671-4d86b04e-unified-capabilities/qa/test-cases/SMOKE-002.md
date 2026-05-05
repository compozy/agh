# SMOKE-002: Create Channel Critical Path

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-23

## Objective

Verify that a user can create a network channel and that the API materializes sessions, durable metadata, and a readable channel detail response.

## Preconditions

- Network is enabled.
- Workspace fixture contains at least two agents.
- Session manager and network store are available.

## Test Steps

1. Submit `POST /api/network/channels` with channel, workspace ID, purpose, and two agent names.
   **Expected:** Response is 201 and includes channel detail with workspace ID, purpose, created-by, sessions, peers, and counts.
2. Read `GET /api/network/channels/{channel}` for the created channel.
   **Expected:** Response is 200 and matches the created channel metadata.
3. Read `GET /api/network/channels`.
   **Expected:** Channel summary includes the created channel, session count, peer count, and last activity.
4. Trigger a readback failure after session creation.
   **Expected:** Created sessions are stopped with failed rollback cause and channel metadata is removed when it had been persisted.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Duplicate agents | `["coder","coder"]` | 400 validation error |
| Missing workspace | empty `workspace_id` | 400 validation error |
| Agent unavailable | unknown agent | session/workspace mapped error and no partial channel |

## Related

- TC-FUNC-008
- TC-INT-101
- TC-UI-003
