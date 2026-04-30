# Tool Registry Canonical Surface Traceability Matrix

**qa-output-path:** `.compozy/tasks/tools-refac`
**Status:** Planning complete, not executed
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

This document maps each implementation task (01-11) to the QA scenarios that prove its invariants and to the regression hot spots task_13 must monitor. Every task is mapped to at least one P0 scenario and one regression hot spot, and every TC is mapped back to its originating task / ADR / TechSpec section.

## Task → Test Cases

### Task 01 — Dynamic Policy Input Resolver and Default Discovery Overlay

| Aspect | Test Cases | Hot Spots |
|--------|------------|-----------|
| Default discovery overlay (`agh__bootstrap`, `agh__catalog`) | TC-FUNC-001, TC-INT-002, TC-INT-003 | Empty-`tools` agents must still see discovery; explicit deny must still win. Regression after `internal/tools/policy.go` or `internal/tools/builtin/toolsets.go` edits. |
| Per-call policy recomputation | TC-FUNC-001, TC-INT-006 | Cache MUST NOT become authority; invalidation keys: agent reload, lineage change, source-health change, hook reload, MCP auth health change, config overlay change. |
| Operator vs session projection divergence | TC-INT-001, TC-INT-005 | Session projection callable-only; operator projection includes denial reason codes. |
| Reason-code taxonomy | TC-INT-001, TC-INT-005, TC-FUNC-004, TC-FUNC-005, TC-FUNC-007, TC-FUNC-008, TC-SEC-004, TC-SEC-005 | Deterministic codes for unavailable, denied, hook-blocked, source-health-blocked, scope-not-allowed. |

### Task 02 — Tools Guidance Assets and Startup Prompt Section

| Aspect | Test Cases | Hot Spots |
|--------|------------|-----------|
| `HarnessPromptSectionTools` rendering at startup | TC-FUNC-002 | Section must order after `skills` and before `network` when `[tools].enabled` is true. Regression after `internal/daemon/composed_assembler.go` or `internal/daemon/prompt_sections.go` edits. |
| Bundled `agh-tools-guide` | TC-FUNC-002, TC-REG-005 | Must teach `agh__tool_search → agh__tool_info → invoke`; CLI is management/fallback. |
| Catalog text references `agh__skill_view` first | TC-FUNC-002, TC-REG-005 | Catalog string in `internal/skills/catalog.go` must not regress to CLI-first wording. |

### Task 03 — Coordination, Session, and Workspace Read Surfaces

| Aspect | Test Cases | Hot Spots |
|--------|------------|-----------|
| `agh__coordination` extension (`network_status`, `network_channels`, `network_inbox`) | TC-FUNC-003, TC-INT-002 | Must extend, not rename; `network_peers`/`network_send` remain available. |
| `agh__sessions` read-only family | TC-FUNC-003, TC-INT-002 | Visibility/`not-found` semantics aligned with CLI/HTTP/UDS. |
| `agh__workspace` read-only family | TC-FUNC-003, TC-INT-002 | Same scope/visibility rules as CLI. |

### Task 04 — Memory, Observe, and Bridge Read Surfaces

| Aspect | Test Cases | Hot Spots |
|--------|------------|-----------|
| `agh__memory` reads | TC-FUNC-003, TC-INT-002, TC-SEC-001 | Scope filter and redaction preserved; no raw secrets; native handler kept in sync with descriptor (see L-???; cf. shared workflow memory). |
| `agh__observe` reads | TC-FUNC-003, TC-INT-002, TC-SEC-001 | Events/metrics/search must omit `claim_token`; `actor_id`/`actor_kind` remain. |
| `agh__bridges` reads | TC-FUNC-003, TC-INT-002 | Bridge provider-config omitted; status only. |

### Task 05 — Config Mutable Tool Family

| Aspect | Test Cases | Hot Spots |
|--------|------------|-----------|
| Allowed paths mutate via tool with same writer/validator as CLI | TC-FUNC-004, TC-INT-002 | Tool, CLI, HTTP, UDS converge. |
| Trust-root denials | TC-FUNC-004, TC-SEC-004 | `[daemon]`, `[http]`, `[permissions]`, sandbox bootstrap, `memory.global_dir`, provider command/env, MCP server transport, `[log]`. |
| Secret denials | TC-FUNC-004, TC-SEC-004 | Provider API keys, MCP auth secret-bearing fields, raw secret writes. |
| Approval gating | TC-FUNC-004, TC-AUT-001 (autonomy approval) | `agh__config_set`/`unset` require mutating approval. |

### Task 06 — Hook Management Tool Family

| Aspect | Test Cases | Hot Spots |
|--------|------------|-----------|
| Hook read surfaces | TC-FUNC-005, TC-INT-002 | Native handler in sync with descriptor; events/runs visible. |
| Source-immutable hook protection | TC-FUNC-005, TC-SEC-005 | `HOOK_SOURCE_IMMUTABLE` for extension- or source-owned declarations; `HookSourceSkill` immutability covered. |
| Mutation reuses normalization & validation | TC-FUNC-005, TC-SEC-005 | `HOOK_VALIDATION_FAILED`, `HOOK_SECRET_INPUT_FORBIDDEN`, `HOOK_APPROVAL_REQUIRED`. |

### Task 07 — Automation Tool Family

| Aspect | Test Cases | Hot Spots |
|--------|------------|-----------|
| Job CRUD + trigger + history | TC-FUNC-006, TC-INT-002 | Tool/CLI/HTTP/UDS converge on the existing automation manager. |
| Trigger CRUD + history | TC-FUNC-006, TC-INT-002 | Same validators. |
| Run inspection | TC-FUNC-006, TC-INT-002 | Persisted run records identical to CLI/HTTP. |
| Approval + denial taxonomy | TC-FUNC-006 | `AUTOMATION_SCOPE_FORBIDDEN`, `AUTOMATION_SECRET_INPUT_FORBIDDEN`, `AUTOMATION_VALIDATION_FAILED`, `AUTOMATION_APPROVAL_REQUIRED`. |

### Task 08 — Extension Lifecycle Tool Family

| Aspect | Test Cases | Hot Spots |
|--------|------------|-----------|
| Search/list/info | TC-FUNC-007, TC-INT-002 | Marketplace + local source distinctions preserved. |
| Install/update/remove + rollback | TC-FUNC-007 | Failure paths roll back registry/disk identically to CLI; managed install reuse. |
| Enable/disable + reconciliation | TC-FUNC-007 | `internal/extension/tool_reconciliation.go` stays consistent. |
| Trust-source + approval | TC-FUNC-007 | `EXTENSION_SOURCE_FORBIDDEN`, `EXTENSION_APPROVAL_REQUIRED`, `EXTENSION_NOT_INSTALLED`, `EXTENSION_VALIDATION_FAILED`. |

### Task 09 — Session-Bound Autonomy Tools and Claim-Token Hard Cut

| Aspect | Test Cases | Hot Spots |
|--------|------------|-----------|
| `agh__autonomy` family routes to `task.Service` writers | TC-AUT-001, TC-AUT-006, TC-INT-002 | Single ownership path; no parallel writer. |
| `run_id`-keyed contracts (no raw `claim_token`) | TC-AUT-001, TC-SEC-001, TC-REG-001, TC-REG-004 | CLI flag deleted, contract DTO deleted, OpenAPI no longer mentions `claim_token` for AGH-owned routes; `web/src/systems/tasks/{types,mocks}` clean. |
| Cross-session denials | TC-AUT-002, TC-AUT-005 | `AUTONOMY_FOREIGN_RUN`, single-success heartbeat. |
| Single-lease invariant per session | TC-AUT-003, TC-AUT-004 | `AUTONOMY_LEASE_ALREADY_HELD`, `AUTONOMY_NO_ACTIVE_LEASE`, `AUTONOMY_LEASE_EXPIRED`. |
| Network surface raw-token rejection | TC-SEC-003 | `network_raw_token_rejected`. |

### Task 10 — MCP Auth Status and Hosted MCP Projection Parity

| Aspect | Test Cases | Hot Spots |
|--------|------------|-----------|
| `agh__mcp_auth_status` redacted diagnostics | TC-FUNC-008, TC-SEC-002 | `token_present` is the only public diagnostic; tokens/PKCE/codes never cross. |
| Login/logout remain operator-only | TC-FUNC-008 | No tool family for `mcp.auth.login`/`logout`. |
| Hosted MCP `tools/list` parity | TC-INT-003 | Set + ordering + reason codes match `GET /api/sessions/{id}/tools`. |
| Approval bridge invariants | TC-INT-004 | `approval_required` + `approval_unreachable`/`approval_timed_out`/`approval_canceled`. |
| Hosted MCP bind safety | TC-SEC-006 | Bind nonce single-use; UDS peer-creds + AGH binary path validation. |

### Task 11 — Site Docs, Generated References, and Example Alignment

| Aspect | Test Cases | Hot Spots |
|--------|------------|-----------|
| `make codegen-check` clean | TC-REG-001 | OpenAPI no longer references `claim_token` for AGH-owned routes; `web/src/generated/agh-openapi.d.ts` regenerated. |
| `make cli-docs` clean | TC-REG-002 | Hand-authored `cli-reference/index.mdx` and `meta.json` keep `tool/` and `toolsets/` listed; `bun run format` re-aligns tables. |
| Site build | TC-REG-003 | `packages/site` build + source tests including `runtime-tools-canonical-docs.test.ts` pass. |
| Web `tasks` regression | TC-REG-004 | `web/src/systems/tasks/types.ts` and `mocks/fixtures.ts` carry no `claim_token`. |
| Stale prose deletions | TC-REG-005 | `agent-md.mdx`, `definitions.mdx`, `task-runs-and-leases.mdx`, `agh-agent-setup/SKILL.md`, `internal/skills/catalog.go` no longer teach CLI-first or opt-in discovery. |

## Test Case → Source Authorities

| Case | TechSpec sections | ADRs | Implementation surfaces |
|------|-------------------|------|-------------------------|
| TC-FUNC-001 | "Implementation Design", "Data-Model Field Rationale", "Test Strategy → Unit Tests" | ADR-001, ADR-002 | `internal/tools/policy.go`, `internal/tools/builtin/toolsets.go`, `internal/daemon/native_tools.go` |
| TC-FUNC-002 | "Skills, Tools, Resources, Bundles", "Architectural Boundaries" | ADR-001 | `internal/daemon/prompt_sections.go`, `internal/daemon/composed_assembler.go`, `internal/skills/catalog.go`, `internal/skills/bundled/skills/agh-tools-guide` |
| TC-FUNC-003 | "Canonical Built-In Surface", "Implementation Steps" | ADR-001, ADR-002 | `internal/tools/builtin/{network,sessions,workspace,memory,observe,bridges}.go` |
| TC-FUNC-004 | "Mutable Surface Policy → agh__config", "Config Lifecycle" | ADR-002, ADR-006 | `internal/tools/builtin/config.go`, `internal/daemon/native_config_hook_tools.go`, `internal/config/{config.go,merge.go,persistence.go,tools.go}` |
| TC-FUNC-005 | "Mutable Surface Policy → agh__hooks", "Hooks" | ADR-002, ADR-006 | `internal/tools/builtin/hooks.go`, `internal/hooks/{normalize.go,permission.go,introspection.go}`, `internal/config/hooks.go` |
| TC-FUNC-006 | "Mutable Surface Policy → agh__automation" | ADR-006 | `internal/tools/builtin/automation.go`, `internal/automation/{manager.go,validate.go,persistence.go}` |
| TC-FUNC-007 | "Mutable Surface Policy → agh__extensions", "Extensibility Integration Plan" | ADR-004, ADR-006 | `internal/tools/builtin/extensions.go`, `internal/extension/{manager.go,registry.go,install_managed.go,tool_reconciliation.go}` |
| TC-FUNC-008 | "Hosted MCP", "Existing MCP Config And Auth" | ADR-004 | `internal/tools/builtin/mcp_auth.go`, `internal/tools/mcp.go`, `internal/mcp/auth/service.go`, `internal/api/core/conversions.go` |
| TC-INT-001 | "Implementation Design", "Test Strategy → Unit Tests" | ADR-002 | `internal/tools/policy.go`, `internal/api/core/tools.go` |
| TC-INT-002 | "API Endpoints", "Agent Manageability Plan" | ADR-001, ADR-002 | `internal/api/core/tools.go`, `internal/api/httpapi`, `internal/api/udsapi`, `internal/cli/tool.go`, `internal/daemon/hosted_mcp.go` |
| TC-INT-003 | "Hosted MCP" | ADR-002 | `internal/daemon/hosted_mcp.go`, `internal/mcp/hosted.go`, `internal/api/core/sessions.go` |
| TC-INT-004 | "Hosted MCP → approval bridge" | ADR-002 | `internal/daemon/tool_approval_bridge.go`, `internal/mcp/hosted.go` |
| TC-INT-005 | "Test Strategy → Unit Tests" | ADR-002 | `internal/hooks/permission.go`, `internal/tools/policy.go` |
| TC-INT-006 | "Known Risks → cache invalidation" | ADR-002 | `internal/tools/policy.go`, `internal/daemon/native_tools.go` |
| TC-SEC-001 | "Safety Invariants", "Monitoring and Observability", "Post-Implementation Residual Checks" | ADR-005 | `internal/api/contract/agents.go`, `internal/api/core/agent_tasks.go`, `internal/cli/task.go`, `internal/api/spec/spec.go`, `internal/network/*`, `internal/observe/*` |
| TC-SEC-002 | "Hosted MCP", "Existing MCP Config And Auth" | ADR-004 | `internal/mcp/auth/service.go`, `internal/tools/builtin/mcp_auth.go` |
| TC-SEC-003 | "Safety Invariants" | ADR-005 | `internal/api/core/network.go`, `internal/cli/network.go`, and `agh__network_send` handler |
| TC-SEC-004 | "Mutable Surface Policy → agh__config" | ADR-006 | `internal/config/persistence.go`, policy helpers in `internal/tools/builtin/config.go` |
| TC-SEC-005 | "Mutable Surface Policy → agh__hooks" | ADR-006 | `internal/hooks/permission.go`, `internal/tools/builtin/hooks.go` |
| TC-SEC-006 | "Hosted MCP → bind nonce + peer creds" | ADR-002 | `internal/daemon/hosted_mcp.go`, `internal/mcp/hosted.go` |
| TC-AUT-001..006 | "Session-Bound Autonomy Lookup", "Bootstrap Task Tools", "Post-Implementation Residual Checks" | ADR-003, ADR-005 | `internal/task/{lease.go,lease_manager.go}`, `internal/tools/builtin/autonomy.go`, `internal/api/core/agent_tasks.go`, `internal/cli/task.go` |
| TC-REG-001 | "Docs And Generated Surfaces" | ADR-005 | `internal/api/spec/spec.go`, `web/src/generated/agh-openapi.d.ts` |
| TC-REG-002 | "Docs And Generated Surfaces" | ADR-001 | `packages/site/content/runtime/cli-reference/`, `Makefile` (`cli-docs`) |
| TC-REG-003 | "Docs And Generated Surfaces" | ADR-001..ADR-006 | `packages/site/content/runtime/core/*`, source tests |
| TC-REG-004 | "Web/Docs Impact" | ADR-005 | `web/src/systems/tasks/types.ts`, `web/src/systems/tasks/mocks/fixtures.ts`, generated OpenAPI |
| TC-REG-005 | "Skills, Tools, Resources, Bundles", "Post-Implementation Residual Checks" | ADR-001 | `internal/skills/catalog.go`, `internal/skills/bundled/skills/agh-agent-setup/SKILL.md`, `packages/site/content/runtime/core/configuration/agent-md.mdx` |
| TC-UI-001 | "Web/Docs Impact" | ADR-005, ADR-006 | `web/src/systems/automation/*`, `web/src/systems/settings/*`, `web/src/systems/tasks/*` |

## Coverage Audit (Plan-Level Self-Check)

This audit satisfies the task_12 unit and integration coverage requirements: every implementation task from 01-11 maps to at least one planned scenario AND one regression hot spot, and the dossier covers tool, CLI, HTTP, UDS, hosted MCP, docs, and downstream web artifact verification.

| Task | Mapped Scenarios | Mapped Hot Spots | Surfaces Covered |
|------|------------------|------------------|------------------|
| 01 | TC-FUNC-001, TC-INT-001, TC-INT-002, TC-INT-003, TC-INT-005, TC-INT-006 | Default discovery overlay, projection vs dispatch parity, cache invalidation, reason-code taxonomy | Tool, CLI, HTTP, UDS, hosted MCP |
| 02 | TC-FUNC-002, TC-REG-005 | Prompt section ordering, bundled `agh-tools-guide`, catalog wording | Daemon prompt assembly, bundled skills |
| 03 | TC-FUNC-003, TC-INT-002 | Coordination extension preserves `network_peers`/`network_send`; sessions/workspace visibility parity | Tool, CLI, HTTP, UDS |
| 04 | TC-FUNC-003, TC-INT-002, TC-SEC-001 | Memory/observe/bridge redaction, descriptor/native handler sync | Tool, CLI, HTTP, UDS |
| 05 | TC-FUNC-004, TC-SEC-004, TC-INT-002 | Trust-root + secret + scope denials, approval gating, parity | Tool, CLI, HTTP, UDS |
| 06 | TC-FUNC-005, TC-SEC-005, TC-INT-002, TC-INT-005 | Source-immutable hooks, secret-input denials, normalization reuse | Tool, CLI, HTTP, UDS |
| 07 | TC-FUNC-006, TC-INT-002 | CRUD + trigger + history parity, approval taxonomy | Tool, CLI, HTTP, UDS |
| 08 | TC-FUNC-007, TC-INT-002 | Marketplace/local-source distinctions, rollback, approval | Tool, CLI, HTTP, UDS |
| 09 | TC-AUT-001..006, TC-SEC-001, TC-SEC-003, TC-INT-002, TC-REG-001, TC-REG-004 | Session-bound contract, foreign run/lease invariants, redaction across surfaces, codegen co-ship | Tool, CLI, HTTP, UDS, hosted MCP, observe, memory, web fixtures, OpenAPI |
| 10 | TC-FUNC-008, TC-SEC-002, TC-SEC-006, TC-INT-003, TC-INT-004 | Status-only diagnostics, redaction, hosted MCP projection + approval bridge + bind nonce | Tool, CLI, HTTP, UDS, hosted MCP, settings UI |
| 11 | TC-REG-001..005 | Codegen drift, CLI docs drift, site build, web tasks regression, deletion of stale prose | OpenAPI, CLI reference, site build, web Vitest |

| Surface Family | Required by Task | Verified by |
|----------------|------------------|-------------|
| Tool dispatch | 01-10 | TC-FUNC-001, TC-INT-002, TC-INT-003, TC-AUT-001..006 |
| CLI parity | 03-10 | TC-INT-002, TC-FUNC-003..008, TC-AUT-001..006 |
| HTTP/UDS parity | 03-10 | TC-INT-002, TC-FUNC-004..008, TC-AUT-001..006 |
| Hosted MCP | 01, 10 | TC-INT-003, TC-INT-004, TC-SEC-006 |
| Codegen / OpenAPI | 09, 11 | TC-REG-001 |
| CLI docs | 11 | TC-REG-002 |
| Site build | 11 | TC-REG-003 |
| Web `tasks` system | 09, 11 | TC-REG-004 |
| Catalog/prompt fidelity | 02, 11 | TC-FUNC-002, TC-REG-005 |
| Concurrency/contention | 09 | TC-AUT-005 |
| Redaction sweep | 04, 09, 10 | TC-SEC-001, TC-SEC-002, TC-SEC-003, TC-SEC-005 |
| Approval gating | 05-10 | TC-FUNC-004..008, TC-INT-004 |
| Cache invalidation | 01 | TC-INT-006 |

If any future fix touches a surface that is not represented in this audit, add a new TC and re-audit before reporting task_13 complete.
