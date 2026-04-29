# TC-SEC-009 — Hosted MCP rejects bind without UDS peer + AGH binary validation

- **Priority:** P0
- **Type:** Security / hosted MCP boundary
- **Trace:** Task 10, ADR-002, Safety Invariants 16, 21

## Objective

Prove the hosted MCP proxy fails closed when UDS peer credentials are unavailable, when peer executable does not match the expected AGH binary, when the bind nonce does not match a live launch record, or when peer OS user differs from the daemon user.

## Preconditions

- Daemon issued a launch record for `session_id = sess_test`, `bind_nonce = BIND_NONCE_v1_TESTONLY`, `expected_binary = /path/to/agh`.
- Foreign-process test harness available (different binary path).

## Test Steps

1. Spawn `agh tool mcp --session sess_test --bind-nonce BIND_NONCE_v1_TESTONLY` from the AGH binary itself.
   - **Expected:** Bind succeeds; proxy serves `tools/list`.
2. Spawn the same command from a different OS user.
   - **Expected:** Daemon rejects bind with deterministic permission error; proxy exits non-zero; no tool projection.
3. Spawn a foreign binary masquerading as the proxy with the correct nonce.
   - **Expected:** Daemon rejects bind because expected-binary path does not match.
4. Run on a platform where peer credentials are unavailable (simulated via test hook).
   - **Expected:** Bind fails closed; session receives no hosted projection on that platform.
5. Use a recycled nonce from a prior session.
   - **Expected:** Daemon rejects (single-use; covered also by TC-SEC-010).

## Edge Cases

- ACP `mcpServers[].args` correctly forwards the bind nonce as an argument; capture and verify the value is treated only as a correlation token, not as a bearer secret.
- A foreign process invoking `agh tool mcp` without a valid nonce receives `permission_denied` with no projection content.

## Automation

- **Target:** Integration
- **Status:** Existing partial; Missing platform-unavailable simulation
- **Command/Spec:** `go test ./internal/mcp -run TestHostedMCPBindValidation`; `go test ./internal/testutil/acpmock -run TestHostedMCPACPInjection`
- **Notes:** Most-critical hosted MCP boundary; failure here breaks the entire session-exposure trust model.
