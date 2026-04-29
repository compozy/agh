# TC-FUNC-011 — Effective policy decision composition

- **Priority:** P0
- **Type:** Functional / policy
- **Trace:** Task 03, ADR-005

## Objective

Prove `EffectiveToolDecision` composes ACP ceiling, session lineage, agent policy, registry policy, source policy, descriptor risk, availability, and hook result deterministically. Reasons identify the denying layer.

## Test Steps

1. ACP `approve-reads`, agent allow, source untrusted, descriptor read-only.
   - **Expected:** `approval_required` reason; layer `source_policy`.
2. ACP `approve-all`, agent allow, source trusted, descriptor read-only.
   - **Expected:** Callable; auto-approved.
3. ACP `approve-all`, agent deny via `deny_tools`.
   - **Expected:** Denied; layer `agent_policy`.
4. ACP `approve-all`, source trusted, registry deny.
   - **Expected:** Denied; layer `registry_policy`.
5. Pre-call hook returns deny.
   - **Expected:** Denied; layer `hook`.
6. Backend unhealthy.
   - **Expected:** `availability` denying layer; `tool_unavailable` error.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestEffectiveDecisionMatrix`
