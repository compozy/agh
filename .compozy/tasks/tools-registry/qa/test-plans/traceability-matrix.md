# Tool Registry — P0 / P1 Traceability Matrix

This matrix proves every P0 and P1 case in the QA test-case set names the exact task, TechSpec invariant, ADR, or other authoritative source it proves. Task 16 must consume this file when claiming surface coverage.

## How To Read

- **Case** — TC-* identifier (file under `qa/test-cases/`).
- **Priority** — P0 (smoke / critical) or P1 (targeted / major).
- **Tasks** — implementation task(s) the case validates.
- **TechSpec** — `_techspec.md` section(s) named.
- **ADR(s)** — accepted decision(s) the case proves.
- **Safety Invariant(s)** — numbered invariants from `_techspec.md` § Safety Invariants.

## P0 Cases

| Case | Title | Tasks | TechSpec section | ADR(s) | Safety Inv. |
|------|-------|-------|------------------|--------|-------------|
| TC-FUNC-001 | Canonical `ToolID` validator | T01 | Data Models / `ToolID` | ADR-007 | — |
| TC-FUNC-011 | Effective decision composition | T03 | Implementation Design / Effective Decision | ADR-005, ADR-006 | 1, 2, 3, 4, 5 |
| TC-FUNC-012 | Operator vs session projection differs | T03 | Agent Manageability / Discovery behavior | ADR-006 | 14 |
| TC-FUNC-016 | Canonical-ID collision | T03, T07 | Integration Points / MCP Sources / Collision rules | ADR-007 | 7 |
| TC-FUNC-021 | `agh__skill_view` real content + budget | T05 | Integration Points / Skills | ADR-004 | 11 |
| TC-FUNC-027 | Dispatch ordering | T04 | Implementation Design / Dispatch | — | 1, 2, 11 |
| TC-FUNC-031 | Result limiter parity across surfaces | T04 | Test Strategy | — | 11 |
| TC-FUNC-045 | Hosted MCP injects only AGH-hosted stdio | T10 | Integration Points / Hosted MCP | ADR-002, ADR-010 | 13 |
| TC-INT-001 | Every path enters `Registry.Call` | T04, T10, T11, T12 | MVP Boundary / Architectural Boundaries | ADR-003 | 1 |
| TC-INT-007 | TS extension publishes executable read-only tool | T07 | Integration Points / Extensions | ADR-001, ADR-008 | 8, 9, 19 |
| TC-INT-009 | Go SDK extension publishes executable tool | T08 | Integration Points / Extensions | ADR-009 | 8, 9, 19 |
| TC-INT-011 | Local stdio MCP call-through | T09 | Integration Points / MCP Sources / MCP Library Adoption | ADR-010, ADR-011 | 19 |
| TC-INT-013 | Hosted `tools/list` = `GET /api/sessions/{id}/tools` | T10, T11 | Integration Points / Hosted MCP | ADR-002 | 13 |
| TC-INT-016 | `make verify` on fresh lab | All | Test Strategy | — | — |
| TC-SEC-001 | `deny-all` blocks all backends | T03, T04 | Implementation Design / Effective Decision | ADR-005 | 1, 2, 4 |
| TC-SEC-002 | `approve-reads` does not auto-approve untrusted | T02, T03 | Config Lifecycle / `trusted_sources` | ADR-005 | 5 |
| TC-SEC-003 | `approve-all` does not bypass denies/lineage/hooks | T03, T04 | Implementation Design / Effective Decision | ADR-005 | 4, 6, 7, 10 |
| TC-SEC-004 | Mutating tool mislabeled `read_only` rejected | T01, T06 | Data Models / Risk fields | ADR-008 | 5 |
| TC-SEC-005 | MCP `Authorization` never crosses `internal/tools` | T09 | Integration Points / Existing MCP Config and Auth | ADR-010 | 12, 20 |
| TC-SEC-006 | At-most-one MCP refresh; no new login | T09 | Integration Points / MCP Library Adoption | ADR-010, ADR-011 | 23 |
| TC-SEC-009 | Hosted MCP UDS peer + binary validation | T10 | Integration Points / Hosted MCP authentication | ADR-002 | 16, 21 |
| TC-SEC-010 | Hosted MCP nonce single-use + TTL + redacted | T02, T10 | Config Lifecycle / `bind_nonce_ttl_seconds` | ADR-002 | 16 |
| TC-SEC-011 | Approval token single-use + bound | T11 | API Endpoints / Approval Token Issuance | ADR-005 | 27 |
| TC-SEC-012 | Hosted MCP rejects client-supplied approval token | T10 | Integration Points / Hosted MCP Approval Bridge | ADR-002, ADR-005 | 17, 21 |
| TC-SEC-013 | Approval token absent from logs/events/SSE/etc. | T11 | API Endpoints / Approval Token Issuance | ADR-005 | 27 |

## P1 Cases

| Case | Title | Tasks | TechSpec section | ADR(s) | Safety Inv. |
|------|-------|-------|------------------|--------|-------------|
| TC-FUNC-002 | Tool pattern grammar | T02, T03 | Config Lifecycle / Tool Pattern Grammar | ADR-007 | — |
| TC-FUNC-003 | `Canonicalize(rawServer, rawTool)` byte-stable | T09 | Integration Points / MCP Sources / Canonicalization | ADR-010 | — |
| TC-FUNC-004 | `id_too_long` reason code | T01 | Data Models / `ToolID` | ADR-007 | — |
| TC-FUNC-006 | `SourceRef` preserves provenance | T01, T09 | Data Models / `SourceRef` | ADR-007 | — |
| TC-FUNC-007 | Empty config defaults | T02 | Config Lifecycle / Global `config.toml` | — | — |
| TC-FUNC-008 | Agent grammar validation | T02 | Config Lifecycle / Agent Definitions | ADR-007 | — |
| TC-FUNC-013 | Child session lineage subset | T03 | Config Lifecycle / Agent Definitions | ADR-005 | 6 |
| TC-FUNC-018 | `agh__tool_list` listing | T05 | Integration Points / Built-ins | ADR-004 | — |
| TC-FUNC-022 | `agh__network_peers` | T05 | Network And Tasks | ADR-004 | — |
| TC-FUNC-023 | `agh__network_send` policy enforcement | T05 | Network And Tasks | ADR-005 | 5 |
| TC-FUNC-024 | Bounded task tools cover ADR-004 set | T05 | Network And Tasks | ADR-004 | 18 |
| TC-FUNC-025 | `agh__task_child_create` lineage | T05 | Network And Tasks | ADR-004 | 18 |
| TC-FUNC-026 | Native risk classification | T05 | Network And Tasks | ADR-004, ADR-005 | 5 |
| TC-FUNC-028 | Schema validation rejects malformed input | T04 | Implementation Design / Dispatch | — | 1 |
| TC-FUNC-029 | Hook deny / patch behavior | T04 | Integration Points / Hooks | — | 10 |
| TC-FUNC-030 | Canonical `tool_id` in hook payloads | T04 | Delete Targets | ADR-007 | 10 |
| TC-FUNC-032 | Cancellation propagation | T04 | Implementation Design / Dispatch | — | — |
| TC-FUNC-033 | Telemetry events with redacted fields | T04 | Monitoring and Observability | — | 11, 12 |
| TC-FUNC-034 | Deterministic error mapping | T04, T11 | API Endpoints / Status codes | — | — |
| TC-FUNC-035 | `MCPCallExecutor` lives in `internal/mcp` | T09 | Architectural Boundaries | ADR-010 | 20 |
| TC-FUNC-036 | MCP descriptor `output_schema` preservation | T09 | Data Models / MCPToolDescriptor | ADR-010 | — |
| TC-FUNC-037 | HTTP/SSE transport selection | T09 | Existing MCP Config / Validation | ADR-010, ADR-011 | 26 |
| TC-FUNC-038 | Local stdio MCP `tools/list` and `tools/call` | T09 | MCP Library Adoption | ADR-010 | — |
| TC-FUNC-040 | `tool.provider` capability negotiation | T07 | Core Interfaces / Extension protocol additions | ADR-001, ADR-008 | 8, 19 |
| TC-FUNC-043 | Public Go SDK builds without `internal/*` | T08 | Architectural Boundaries / SDK | ADR-009 | — |
| TC-FUNC-044 | Go SDK digest parity | T08 | Data Models / Schema digest contract | ADR-008 | — |
| TC-FUNC-046 | Hosted MCP raw schema bytes | T10 | MCP Library Adoption / Hosted MCP | ADR-011 | 22 |
| TC-FUNC-047 | Hosted MCP projection stream | T10 | Integration Points / Hosted MCP lifecycle | ADR-002 | 24 |
| TC-FUNC-048 | Approval bridge timeout | T10 | Integration Points / Hosted MCP Approval Bridge | ADR-005 | 17, 25 |
| TC-FUNC-049 | Proxy disconnect cancellation | T10 | Integration Points / Hosted MCP | — | 17 |
| TC-FUNC-050 | HTTP routes return canonical contracts | T11 | API Endpoints | — | — |
| TC-FUNC-051 | UDS / HTTP parity | T11 | API Endpoints / Agent Manageability | — | — |
| TC-FUNC-052 | OpenAPI / web TS codegen sync | T11 | Architectural Boundaries / Co-ship rule | — | — |
| TC-FUNC-053 | `agh tool list -o json` rendering | T12 | Agent Manageability / CLI | ADR-007 | — |
| TC-FUNC-054 | `agh tool info <id>` errors | T12 | Agent Manageability | — | — |
| TC-FUNC-055 | `agh tool invoke` validation + redaction | T12 | Agent Manageability | — | 11 |
| TC-FUNC-057 | Site docs canonical `ToolID` only | T14 | Delete Targets | ADR-007, ADR-008 | — |
| TC-FUNC-058 | Generated CLI/API references regen cleanly | T14 | Docs And Generated Surfaces | — | — |
| TC-INT-002 | Operator vs session projection cross-surface | T03, T11, T12 | Agent Manageability | ADR-006 | — |
| TC-INT-004 | `make codegen` co-ships OpenAPI + TS | T11 | Architectural Boundaries | — | — |
| TC-INT-005 | E2E native tool dispatch | T05, T11, T12 | Test Strategy / E2E | — | 1 |
| TC-INT-012 | Remote OAuth MCP call-through | T09 | Integration Points / MCP | ADR-010 | 12, 20 |
| TC-INT-014 | Hosted MCP safe built-in call | T10, T11 | Integration Points / Hosted MCP | — | 13 |
| TC-PERF-001 | Concurrent dispatch race-free | T04 | Implementation Design / Dispatch | — | 1, 11 |
| TC-SEC-007 | MCP auth status mapping | T09 | Integration Points / MCP auth | ADR-010 | 20 |
| TC-SEC-008 | `cloneDaemonMCPServer` preservation | T09 | Integration Points / Existing MCP Config | ADR-010 | — |
| TC-SEC-014 | Hook payload + result envelope redaction | T04, T06 | Monitoring and Observability | — | 11, 12 |
| TC-UI-001 | Tools list renders canonical IDs | T13 | Impact Analysis / web | ADR-006, ADR-007 | — |
| TC-UI-002 | Tool detail shows redacted MCP auth | T13 | Impact Analysis / web | ADR-010 | 20 |
| TC-UI-003 | No invented login/approval/invoke controls | T13 | Impact Analysis / web | ADR-006 | — |

## P2/P3 Cases (informational only)

P2 cases (TC-FUNC-005, 009, 010, 014, 015, 017, 019, 020, 039, 041, 042, 056; TC-INT-003, 006, 008, 010, 015; TC-PERF-002, 003; TC-UI-004, 005, 006) and P3 cases (covered through full-regression sampling) trace to the same techspec sections; explicit listing is omitted because exit criteria do not require P2/P3 to be 100%.

## Change Control

Any new task / ADR / safety invariant added after 2026-04-29 that affects tool-registry behavior must add a new row in this matrix and a corresponding `TC-*` case before merging.

## References

- `_techspec.md` § Safety Invariants (1-27)
- `adrs/adr-001..011`
- `task_01..task_14` task files and `_tasks.md`
