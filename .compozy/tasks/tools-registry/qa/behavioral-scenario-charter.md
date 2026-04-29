# Tool Registry Task 16 Behavioral Scenario Charter

## Scenario

AGH is used by a small startup preparing a launch review. The operator needs to inspect the available tool registry, expose only callable tools to an agent session, validate extension-authored tools, call an external MCP fixture through AGH-owned policy and redaction boundaries, and confirm web diagnostics explain registry state without inventing unsupported actions.

## Operator Intent

- Confirm native AGH tools, extension-host tools, and MCP-backed tools are discoverable through CLI, HTTP, UDS, and hosted MCP.
- Confirm session-visible tools are narrower than operator diagnostics and match policy, approval, auth, and lineage.
- Confirm approval-required and denied calls fail closed with deterministic reason codes.
- Confirm token, nonce, and sensitive input sentinels never leak into logs, traces, browser evidence, events, CLI JSON, HTTP JSON, UDS JSON, or MCP responses.

## Business Outcome

The launch operator can decide whether Tool Registry is safe to enable for agent sessions because every executable backend has real dispatch evidence, every diagnostic surface is redacted and explainable, and no P0/P1 defect remains open.

## Agent Cast

- Operator agent: uses CLI/HTTP/UDS to inspect and invoke tools.
- QA agent: runs smoke, targeted, full, security, and web diagnostics flows.
- Extension author agent: validates TypeScript and Go extension-host fixtures and manifest/runtime reconciliation.
- Runtime security agent: validates MCP auth, hosted MCP bind, approval token, and redaction boundaries.

## Live Provider Plan

Provider-backed commands must use `PROVIDER_HOME` and `PROVIDER_CODEX_HOME` from `.compozy/tasks/tools-registry/qa/bootstrap-manifest.json`. If live provider credentials are unavailable in the isolated provider home, Task 16 will record that boundary and validate every reachable daemon, CLI, HTTP, UDS, SDK, MCP fixture, browser, and docs/build surface instead of substituting mocks as final proof.

## Disruption Probes

- Denied policy path before happy-path invocation.
- Conflicted canonical tool ID hidden from session projection but visible to operator diagnostics.
- Hosted MCP bind without valid peer/binary/nonce rejected.
- Approval token replay rejected after first use.
- Remote MCP auth failure maps to redacted deterministic reason codes.
- Web diagnostics show unavailable/auth-required/conflicted states without login, approval, or invoke controls that the daemon does not support.
