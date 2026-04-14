# TC-SEC-005: Marketplace Extension Capability Ceiling

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Security |
| **Estimated Time** | 3 min |
| **Module** | `internal/extension/capability.go` |
| **OWASP** | A01:2021 — Broken Access Control |

## Objective

Validate that marketplace-installed extensions receive restricted capabilities only, not full trust-level permissions.

## Preconditions

- Extension installed via marketplace (source = `SourceMarketplace`).

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Install extension from marketplace | **Expected:** Extension registered with `SourceMarketplace` source. |
| 2 | Query extension's allowed capabilities | **Expected:** Only `memory.read`, `observe.read`, `session.read`, `skills.read`, `tool.read`. |
| 3 | Attempt to use a capability not in the allowed list (e.g., `session.write`) | **Expected:** Capability denied. |
| 4 | Compare with locally-installed extension capabilities | **Expected:** Local extensions have broader capabilities. |

## Edge Cases

- Extension manifest declares capabilities it shouldn't have: should be capped by source trust level.
- Extension upgraded from local to marketplace: capabilities should be reduced.
