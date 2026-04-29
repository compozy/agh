# Tool Registry — Security & Redaction Regression Suite

- **Lane:** Security / Redaction (45-90 min)
- **Frequency:** Per change to `internal/mcp`, `internal/mcp/auth`, hosted MCP code, dispatch/result limiter, hooks, observability, settings/diagnostics, or any file emitting tool-call telemetry; weekly otherwise; mandatory pre-release.
- **Coverage:** All `TC-SEC-*` cases + redaction-relevant sub-cases pulled from functional/integration suites.
- **Stop rule:** Any sentinel leak halts execution and produces a `BUG-NNN.md` with Severity = Critical.

## Sentinels (must NOT appear in any artifact)

These literal byte sequences are seeded into fixtures so leak detection is deterministic. Sentinels live in `qa/fixtures/redaction-sentinels.json` (created by Task 16).

| Sentinel | Source | Where it must NEVER appear |
|----------|--------|----------------------------|
| `mcp:test:bearer:OAUTHTOKEN_v1` | Fake OAuth issuer access token | `internal/tools` errors, registry events, tool results, CLI JSON, HTTP JSON, UDS JSON, MCP responses, SSE/event payloads, daemon logs, settings output, web payloads, generated docs |
| `mcp:test:refresh:REFRESHTOKEN_v1` | Fake OAuth issuer refresh token | Same locations |
| `mcp:test:authcode:AUTHCODE_v1` | Fake OAuth authorization code | Same locations |
| `mcp:test:pkce:VERIFIER_v1` | Fake PKCE verifier | Same locations |
| `BIND_NONCE_v1_TESTONLY` | Hosted MCP bind nonce | ACP payload diagnostics, process diagnostics, crash bundles, logs, SSE/events, settings output, tool results, MCP responses (it MAY appear redacted as a correlation id only) |
| `APPROVAL_TOKEN_v1_TESTONLY` | CLI/HTTP/UDS approval token | Logs, events, SSE, hosted MCP, persisted state, diagnostics, web payloads (it MAY appear only in the authenticated issuance response and the matching invoke request) |
| `CLAIM_TOKEN_v1_TESTONLY` | Task claim token | All tool inputs/outputs persisted or emitted by AGH-owned surfaces |
| `tools.sensitive.field:LEAK_v1` | Tool input field marked sensitive | Tool result envelopes after redaction, hook payloads, events, telemetry |

## Execution Order

1. **Sentinel seeding.** Inject all sentinels via fixture harness. Confirm fixtures contain them.
2. **Negative isolation cases.** Run TC-SEC-009/010 (hosted MCP bind validation) and TC-SEC-001..004 (policy denies) before any happy path.
3. **MCP credential boundary.** Run TC-SEC-005..008.
4. **Approval and bind nonce isolation.** Run TC-SEC-011..013.
5. **Redaction in hooks/observability.** Run TC-SEC-014.
6. **Sentinel scan.** Greppable scan across all `qa/logs/**`, `qa/traces/**`, `qa/screenshots/**`, captured CLI/HTTP/UDS JSON, and daemon process diagnostics. Failure on any match.

## Required Cases

| ID | Title | Priority | Sentinel(s) covered |
|----|-------|----------|---------------------|
| TC-SEC-001 | `deny-all` blocks every executable backend at dispatch time | P0 | — |
| TC-SEC-002 | `approve-reads` does not auto-approve untrusted external read-only tools | P0 | — |
| TC-SEC-003 | `approve-all` does not bypass explicit denies, lineage, or hooks | P0 | — |
| TC-SEC-004 | Mutating tool mislabeled `read_only` is rejected/denied at descriptor validation | P0 | — |
| TC-SEC-005 | Remote MCP `Authorization` header never visible to `internal/tools` | P0 | OAUTHTOKEN_v1 |
| TC-SEC-006 | Remote MCP refresh path attempts at most one refresh, never bootstraps a new login | P0 | OAUTHTOKEN_v1, REFRESHTOKEN_v1 |
| TC-SEC-007 | MCP `unconfigured`/`needs_login`/`expired`/`invalid` map to deterministic redacted reason codes | P1 | OAUTHTOKEN_v1 |
| TC-SEC-008 | `cloneDaemonMCPServer` preserves `Transport`/`URL`/`Auth` and never strips OAuth metadata | P1 | — |
| TC-SEC-009 | Hosted MCP rejects bind without UDS peer + AGH binary validation | P0 | BIND_NONCE_v1 |
| TC-SEC-010 | Hosted MCP bind nonce is single-use, TTL-bounded, redacted in logs | P0 | BIND_NONCE_v1 |
| TC-SEC-011 | CLI/HTTP/UDS approval token is single-use and bound to tool/session/workspace/input | P0 | APPROVAL_TOKEN_v1 |
| TC-SEC-012 | Hosted MCP rejects client-supplied approval tokens; uses approval bridge only | P0 | APPROVAL_TOKEN_v1 |
| TC-SEC-013 | Approval token absent from logs, events, SSE, hosted MCP, persisted state, diagnostics | P0 | APPROVAL_TOKEN_v1 |
| TC-SEC-014 | Hook payloads, events, and result envelopes redact `tools.sensitive.field` markings | P1 | tools.sensitive.field |

## Pass / Fail / Conditional

- **PASS:** All P0 pass, all P1 pass, no sentinel match in any artifact.
- **FAIL:** Any P0 fails OR any sentinel match OR any redaction-related P1 fails.
- **CONDITIONAL:** Not allowed for this lane.

## Outputs

- `qa/logs/security/<TC-ID>.log`
- `qa/issues/BUG-NNN.md` with Severity = Critical for any sentinel leak
- `qa/verification-report.md` records the final sentinel-scan command and exit code

## Sentinel Scan Command (Task 16 must run)

```sh
grep -REIn \
    -e 'mcp:test:bearer:' \
    -e 'mcp:test:refresh:' \
    -e 'mcp:test:authcode:' \
    -e 'mcp:test:pkce:' \
    -e 'BIND_NONCE_v1_TESTONLY' \
    -e 'APPROVAL_TOKEN_v1_TESTONLY' \
    -e 'CLAIM_TOKEN_v1_TESTONLY' \
    -e 'tools.sensitive.field:LEAK_v1' \
    .compozy/tasks/tools-registry/qa/logs \
    .compozy/tasks/tools-registry/qa/traces \
    .compozy/tasks/tools-registry/qa/screenshots
```

A non-zero exit (no matches) is the pass condition. Any match is a Critical defect.
