# Daytona SSH Non-PTY Validation

Date: 2026-04-16

## Status

Pending live validation.

The validation harness exists in `internal/environment/daytona/ssh_validation_test.go`, but the current execution
environment does not have `DAYTONA_API_KEY` set. The live gate has not been proven yet. Do not treat SSH as approved
for Task 06 until the credentialed integration test passes.

## How to Run

```bash
DAYTONA_API_KEY=... go test -tags integration ./internal/environment/daytona -run TestDaytonaSSHNonPTYValidation -count=1 -v
```

Optional environment overrides:

- `DAYTONA_API_URL`: Daytona API base URL. Defaults to `https://app.daytona.io/api`.
- `DAYTONA_ORGANIZATION_ID`: optional organization header for JWT-backed Daytona accounts.
- `DAYTONA_SSH_HOST`: SSH gateway host. Defaults to `ssh.app.daytona.io`.

## Validation Scope

The test performs the required blocking gate:

- Creates a Daytona sandbox with `POST /api/sandbox`.
- Requests SSH access with `POST /api/sandbox/{id}/ssh-access?expiresInMinutes=60`.
- Connects through OpenSSH with explicit non-PTY mode (`-T`, no `-t`) and streams through remote `cat`.
- Verifies byte-for-byte stdout for:
  - 100B JSON payload.
  - 10KB JSON payload.
  - 100KB JSON payload.
  - newline-delimited JSON messages.
- Measures a ready-session 1KB payload round trip and fails above `100ms`.
- Fails on terminal artifacts: ANSI escape bytes, carriage returns, backspaces, delete bytes, extra stdout bytes, or mismatched payload bytes.
- Attempts sandbox deletion in test cleanup.

## Current Results

No live Daytona sandbox was created in this run because `DAYTONA_API_KEY` is missing.

| Check                   | Result  | Evidence                  |
| ----------------------- | ------- | ------------------------- |
| Sandbox create          | Not run | Missing `DAYTONA_API_KEY` |
| SSH token create        | Not run | Missing `DAYTONA_API_KEY` |
| Non-PTY SSH connect     | Not run | Missing `DAYTONA_API_KEY` |
| 100B JSON byte match    | Not run | Missing `DAYTONA_API_KEY` |
| 10KB JSON byte match    | Not run | Missing `DAYTONA_API_KEY` |
| 100KB JSON byte match   | Not run | Missing `DAYTONA_API_KEY` |
| NDJSON byte match       | Not run | Missing `DAYTONA_API_KEY` |
| Artifact detection      | Not run | Missing `DAYTONA_API_KEY` |
| 1KB latency under 100ms | Not run | Missing `DAYTONA_API_KEY` |
| Sandbox cleanup         | Not run | No sandbox was created    |

## Gate Decision

Blocked pending credentialed evidence.

Task 06 must not proceed on the assumption that Daytona SSH is clean until:

1. `DAYTONA_API_KEY=... go test -tags integration ./internal/environment/daytona -run TestDaytonaSSHNonPTYValidation -count=1 -v` exits 0.
2. This report is updated with the observed payload latencies and artifact result.

## Fallback Plan

If the credentialed validation fails because Daytona forces PTY allocation, injects terminal artifacts, corrupts byte
streams, or cannot keep 1KB ready-session round-trip latency under 100ms, Task 06 should switch to the ADR-002
WebSocket sidecar path:

- Bake or upload a small Go sidecar into the sandbox.
- The sidecar starts the ACP agent with `os/exec` and clean stdin/stdout pipes.
- AGH connects to the sidecar through Daytona preview/WebSocket routing.
- The provider-internal transport interface remains the boundary, so session, ACP, and ToolHost public contracts do not
  change.

## References

- `.compozy/tasks/sandbox/adrs/adr-002.md`
- Daytona SSH access docs: `https://www.daytona.io/docs/en/ssh-access/`
- Daytona API reference: `https://www.daytona.io/docs/tools/api/`
