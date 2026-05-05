# TC-FUNC-008: Create Channel Validation And Rollback

**Priority:** P0
**Type:** Functional
**Module:** API Core + Session
**Requirement:** Channel creation must be atomic from the operator perspective.

## Objective

Verify create-channel validation, workspace/agent resolution, session creation, metadata persistence, detail readback, and rollback on partial failure.

## Preconditions

- Workspace service can resolve a workspace with known agents.
- Session manager can create sessions and stop them with a cause.
- Network store can write and delete channel metadata.

## Test Steps

1. Submit a valid create-channel request.
   **Expected:** Sessions are created for each selected agent and response is 201 with channel detail.
2. Submit duplicate agent names.
   **Expected:** Response is 400 and no session is created.
3. Submit an unknown agent for the workspace.
   **Expected:** Response maps to session/workspace error and no metadata is written.
4. Make the second session creation fail.
   **Expected:** First session is stopped with failed rollback cause.
5. Make metadata write succeed but detail readback fail.
   **Expected:** Created sessions are stopped and metadata is deleted.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Blank purpose | whitespace | 400 validation error |
| Invalid channel | unsupported characters | 400 network validation error |
| Workspace root missing | workspace resolver error | workspace error status |

## Related

- SMOKE-002
- TC-INT-101
- TC-UI-003
