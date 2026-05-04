# Redaction Sweep Procedure

**qa-output-path:** `.compozy/tasks/tools-refac`
**Status:** Planning complete, not executed
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

The autonomy hard cut and the broadened built-in tool surface make redaction load-bearing across many channels. This document defines the cross-channel sweep procedure and the deterministic patterns task_13 must search for. It is referenced by TC-SEC-001, TC-SEC-002, TC-SEC-003, TC-SEC-005, and any TC that touches autonomy, MCP auth, hooks, or network surfaces.

## Forbidden Patterns (must produce zero matches across the listed channels)

| Pattern | Surfaces that MUST be clean | Allowed exceptions |
|---------|-----------------------------|--------------------|
| `claim_token` (raw value) | CLI human + `-o json` output, HTTP/UDS request/response bodies, hosted MCP request/response bodies, SSE events, daemon logs, observe events, memory entries, web fixtures (`web/src/systems/tasks/mocks/fixtures.ts`), generated OpenAPI (`openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`), site docs | Internal Go state inside `internal/task/lease.go` and writers (server-side only); `claim_token_hash` everywhere as observability metadata |
| `--claim-token` flag | CLI command help/output, site `cli-reference/task/{next,heartbeat,complete,fail,release}.mdx`, generated CLI docs JSON | None |
| `ClaimToken` (Go struct field) | `internal/api/contract/agents.go`, public DTOs, OpenAPI schemas | Internal `task` package types only |
| OAuth tokens / refresh tokens | MCP auth status responses, settings JSON, CLI human/JSON output, SSE, observe events, daemon logs | `internal/mcp/auth/` token store only |
| OAuth `code` (authorization code) | All public surfaces and logs | None |
| PKCE verifier / challenge | All public surfaces and logs | Internal MCP auth flow only |
| Provider API keys | Config JSON via tool/CLI/HTTP/UDS, settings UI payload, logs | None |
| MCP `auth.*` secret-bearing fields | Tool `agh__config_get`/`show` output for affected paths, HTTP/UDS, settings JSON, logs | None (must be denied with `CONFIG_SECRET_PATH_FORBIDDEN`) |
| Provider command env binding | Tool `agh__config_get`/`show` output, HTTP/UDS, settings JSON, logs | Operator surfaces only |
| Webhook secret material (automation) | Automation tool output, CLI/HTTP/UDS, observe events, logs | Internal automation manager only |

## Sweep Targets (channels to scan during execution)

For each TC that triggers a write or read of a sensitive surface, capture the corresponding channel artifact under `qa/logs/<TC-ID>/` and run the grep listed here.

### Channel: CLI human output

```bash
agh task next 2>&1 | tee qa/logs/<TC-ID>/cli-task-next.txt
agh task heartbeat <run_id> 2>&1 | tee qa/logs/<TC-ID>/cli-task-heartbeat.txt
agh mcp auth status 2>&1 | tee qa/logs/<TC-ID>/cli-mcp-auth-status.txt
```

Then:

```bash
grep -nE "claim_token|access_token|refresh_token|pkce|code=|client_secret" qa/logs/<TC-ID>/cli-*.txt
```

Expected: zero matches except optional `claim_token_hash`.

### Channel: CLI JSON output

```bash
agh task next -o json | tee qa/logs/<TC-ID>/cli-task-next.json
agh tool invoke agh__task_run_claim_next -o json | tee qa/logs/<TC-ID>/tool-claim.json
agh tool invoke agh__mcp_auth_status -o json --input '{"server_name":"foo"}' | tee qa/logs/<TC-ID>/tool-mcp-auth.json
```

```bash
jq -r '..|strings' qa/logs/<TC-ID>/*.json | grep -E "claim_token|access_token|refresh_token|pkce|code=|client_secret"
```

### Channel: HTTP/UDS bodies

Capture request/response bodies via `curl -v --unix-socket "$AGH_HOME/run/sock/uds.sock"` and tee to `qa/logs/<TC-ID>/uds-*.txt`. Also exercise the corresponding HTTP routes when the daemon is configured to expose them.

```bash
grep -nE "claim_token|access_token|refresh_token|pkce|code=|client_secret" qa/logs/<TC-ID>/uds-*.txt qa/logs/<TC-ID>/http-*.txt
```

### Channel: SSE events

Open an SSE stream against the daemon while running TC-AUT-001 and TC-AUT-002 and tee the events:

```bash
curl -N --unix-socket "$AGH_HOME/run/sock/uds.sock" http://localhost/api/sessions/$SID/events | tee qa/logs/<TC-ID>/sse-events.txt
```

```bash
grep -nE "claim_token|access_token|refresh_token|pkce|code=|client_secret" qa/logs/<TC-ID>/sse-events.txt
```

### Channel: Daemon logs

```bash
cat $AGH_HOME/logs/daemon.log | tee qa/logs/<TC-ID>/daemon.log
grep -nE "claim_token|access_token|refresh_token|pkce|code=|client_secret" qa/logs/<TC-ID>/daemon.log
```

### Channel: Observe events / memory entries

```bash
agh observe events -o json | tee qa/logs/<TC-ID>/observe-events.json
agh memory list -o json | tee qa/logs/<TC-ID>/memory-list.json
```

```bash
grep -nE "claim_token|access_token|refresh_token|pkce|code=|client_secret" qa/logs/<TC-ID>/observe-events.json qa/logs/<TC-ID>/memory-list.json
```

### Channel: Hosted MCP

Record the hosted MCP `tools/list` and `tools/call` JSON-RPC frames during TC-AUT-001 / TC-INT-003:

```bash
# the bind capture script tees stdio to this file
cat qa/logs/<TC-ID>/hosted-mcp-frames.jsonl
grep -nE "claim_token|access_token|refresh_token|pkce|code=|client_secret" qa/logs/<TC-ID>/hosted-mcp-frames.jsonl
```

### Channel: Generated artifacts

```bash
grep -nE "\"claim_token\"" openapi/agh.json
grep -nE "claim_token" web/src/generated/agh-openapi.d.ts
grep -nE "claim_token" web/src/systems/tasks/types.ts
grep -nE "claim_token" web/src/systems/tasks/mocks/fixtures.ts
```

Allowed match: `claim_token_hash` only.

### Channel: Site docs

```bash
grep -RIn "--claim-token" packages/site/content/runtime/cli-reference/task
grep -RIn "claim_token" packages/site/content/runtime/core
```

Allowed: `claim_token_hash` references in observability docs.

## Procedure (per TC requiring redaction sweep)

1. Identify which channels the TC touches. Reference the per-TC "Channels exercised" section.
2. Run the TC scenarios and tee output to `qa/logs/<TC-ID>/`.
3. Run the corresponding grep commands above. Each grep must produce zero unexpected matches.
4. If a grep finds `claim_token_hash`, confirm the TC explicitly asserts `claim_token_hash` is present (observability) and not `claim_token`.
5. If a grep finds anything else, file `BUG-NNN.md` with:
   - The TC ID.
   - The exact channel and line.
   - The extracted secret pattern (in redacted form for the bug record).
   - The expected redaction location (which writer/log/observe path should have redacted it).
6. Fix the root cause, regenerate evidence, re-run the sweep, and link the fix in the TC log.

## Coordination With Other Suites

- TC-SEC-001 owns the autonomy `claim_token` sweep across every channel above.
- TC-SEC-002 owns the MCP auth status sweep (token / code / PKCE / callback secrets).
- TC-SEC-003 owns the `agh__network_send` raw-token rejection sweep on the message-body and message-metadata levels.
- TC-SEC-004 owns config trust-root/secret denial sweep (and indirectly redaction by way of `agh__config_get`/`show` output assertions).
- TC-SEC-005 owns hook secret-input rejection sweep.

A redaction failure on any channel is a P0 stop-condition for the whole regression run.
