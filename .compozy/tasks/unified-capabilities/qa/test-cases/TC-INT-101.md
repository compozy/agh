# TC-INT-101: Daemon To Session To Store Channel Creation

**Priority:** P0
**Type:** Integration
**Systems:** API Core, Session Manager, Workspace Service, Global DB, Network Runtime
**API Endpoint:** `POST /api/network/channels`

## Objective

Verify data and side effects flow correctly when the daemon creates a network channel through real or high-fidelity local components.

## Preconditions

- Temporary AGH home and workspace fixtures are available.
- Workspace fixture has at least two agents.
- Global DB is backed by `t.TempDir()`.

## Test Steps

1. Start a local handler or daemon fixture with network enabled.
   **Expected:** Network status is enabled and service dependencies are present.
2. POST a create-channel request for two agents.
   **Expected:** API returns 201 and creates one session per agent.
3. Read global session index and network channel metadata.
   **Expected:** Sessions have the requested channel and channel metadata is persisted once.
4. Read channel detail and channel list through public API.
   **Expected:** API outputs match persisted state.
5. Force a downstream failure and retry from clean state.
   **Expected:** No orphaned sessions or channel metadata remain.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Empty agent catalog | no agents | 400 and no store writes |
| Session creation error after first success | injected error | rollback stops created session |
| Metadata write error | store error | sessions rolled back |

## Related

- SMOKE-002
- TC-FUNC-008
