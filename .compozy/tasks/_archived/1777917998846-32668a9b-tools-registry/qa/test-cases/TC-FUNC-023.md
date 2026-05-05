# TC-FUNC-023 — `agh__network_send` enforces channel/session policy

- **Priority:** P1
- **Type:** Functional / mutating native tool
- **Trace:** Task 05, ADR-005

## Objective

Prove `agh__network_send` is a mutating, open-world tool that requires explicit grant and approval; cannot auto-approve under `approve-reads`.

## Test Steps

1. `permissions.mode = "approve-reads"`, no source grant.
   - **Expected:** Hidden from session projection; operator view shows reason `policy_denied` / `approval_required`.
2. With explicit allow + approval.
   - **Expected:** Call goes through network manager; channel/session policy applies.
3. Invoking a peer that the session lineage does not allow.
   - **Expected:** `policy_denied` from network service, mapped to `tool_denied` reason.
4. `read_only = false`, `destructive = false`, `open_world = true` flags asserted on descriptor.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestNativeNetworkSend`
