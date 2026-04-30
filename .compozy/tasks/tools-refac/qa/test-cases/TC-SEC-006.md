# TC-SEC-006: Hosted MCP Bind Nonce, UDS Peer Credentials, And AGH Binary Path Validation

**Priority:** P0 (Critical)
**Type:** Security / Redaction
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the hosted MCP bind path validates all of: a fresh single-use launch nonce, the UDS peer's effective UID, and the AGH binary path. Confirm foreign or stale bind attempts fail closed, the launch record is invalidated on first use, and the same UDS connection cannot be re-bound to a foreign session/workspace mid-flight.

## Traceability

- Task: task_10.
- TechSpec: "Hosted MCP" (bind validation block).
- ADR: ADR-002.
- Surfaces: `internal/daemon/hosted_mcp.go`, `internal/mcp/hosted.go`.

## Preconditions

- Isolated `AGH_HOME` from `agh-qa-bootstrap`.
- A test harness that can spawn a foreign local process (different UID or different binary path) and connect to the daemon UDS socket.
- A test seam that exposes nonce minting and TTL.

## Test Steps

1. **Happy path:**
   ```bash
   NONCE=$(agh internal mint-mcp-nonce --session $SID -o json | jq -r .nonce)
   agh tool mcp --session $SID --bind-nonce $NONCE 2>&1 | tee qa/logs/TC-SEC-006/bind-good.log
   ```
   - **Expected:** Bind succeeds; daemon log shows the launch record matched. Subsequent `tools/list` works.

2. **Replay (single-use):**
   ```bash
   agh tool mcp --session $SID --bind-nonce $NONCE 2>&1 | tee qa/logs/TC-SEC-006/bind-replay.log
   ```
   - **Expected:** Second bind with the same nonce fails with deterministic permission error; daemon log confirms the launch record was invalidated on the first successful bind.

3. **Expired nonce:**
   - Mint a nonce, wait past TTL, attempt bind.
   - **Expected:** Bind fails closed; reason `bind_nonce_expired` (or canonical equivalent).

4. **Foreign UID:**
   - From a process running as a different UID, attempt to connect to the same UDS socket and bind with a freshly minted nonce.
   - **Expected:** Daemon rejects because peer UID does not match. No projection returned.

5. **Foreign binary path:**
   - Launch a binary at a path that does not match the expected AGH binary; supply a valid nonce.
   - **Expected:** Daemon rejects because peer executable mismatch. No projection returned.

6. **Peer-credentials unavailable:**
   - On a platform/build where peer-cred lookup is unsupported, attempt bind.
   - **Expected:** Hosted MCP fails closed. Session does not receive a hosted registry projection.

7. **Session/workspace switch attempt mid-bind:**
   - After successful bind, attempt to call `tools/list` with a forged `session_id` or `workspace_id` header on the same UDS connection.
   - **Expected:** Daemon ignores client-supplied identifiers and continues to project the originally bound session. Forged-id call returns deterministic permission error or normalizes back to the bound session, depending on daemon design — record actual behavior and confirm it matches the TechSpec.

8. Run focused Go tests:
   ```bash
   go test ./internal/daemon -run "TestHostedMCPBind|TestNonce|TestPeerCred" -count=1 \
     | tee qa/logs/TC-SEC-006/daemon-tests.log
   go test ./internal/mcp -run "TestHosted" -count=1 | tee qa/logs/TC-SEC-006/mcp-tests.log
   ```

## Evidence To Capture

- Bind logs for happy/replay/expired/foreign-uid/foreign-binary cases.
- Daemon log filtered for `mcp_bind` events.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Nonce reused after session end | session terminated, nonce reused | Bind rejected (record invalidated) |
| Connection drops mid-call | proxy disconnect | Bind invalidated; subsequent calls require re-bind |
| Multiple nonces minted for the same session | only one consumed | Outstanding nonces remain usable until TTL or session end |

## Channels Exercised

- Daemon UDS bind path.
- Hosted MCP transport.

## Related Test Cases

- TC-INT-003 (hosted MCP `tools/list` parity).
- TC-INT-004 (approval bridge).
