# TC-FUNC-012: Grant computation intersection

**Priority:** P1
**Type:** Functional
**Package:** internal/extension
**Related Tasks:** 04

## Objective

Validate that the effective grant for an extension is computed as the intersection of five independent policy layers: (1) surface legality (which kinds a surface allows), (2) source-tier ceiling (maximum permissions for the extension's trust tier), (3) operator configuration (admin overrides), (4) manifest request (what the extension asks for), and (5) session-mode narrowing (runtime restrictions based on session type). The final grant must be the narrowest subset that satisfies all five constraints simultaneously.

## Preconditions

- Surface `agent` allows kinds: `["tool", "skill", "mcp_server", "automation.job"]`.
- Source tier `community` has ceiling: `["tool", "skill", "mcp_server"]` (no `automation.job`).
- Operator config grants extension `ext-X`: `["tool", "skill"]` (further restricted).
- Extension `ext-X` manifest requests: `["tool", "skill", "mcp_server", "automation.job"]`.
- Session mode `sandbox` narrows to: `["tool"]` only.

## Test Steps

1. Compute the effective grant for `ext-X` in a normal (non-sandbox) session.
   **Expected:** 
   - Surface legality passes: `["tool", "skill", "mcp_server", "automation.job"]`.
   - Source-tier ceiling narrows to: `["tool", "skill", "mcp_server"]` (removes `automation.job`).
   - Operator config narrows to: `["tool", "skill"]` (removes `mcp_server`).
   - Manifest request intersects: `["tool", "skill"]` (manifest requested all, but only these survive).
   - No session-mode narrowing in normal mode.
   - **Final grant: `["tool", "skill"]`.**

2. Compute the effective grant for `ext-X` in a `sandbox` session.
   **Expected:**
   - Steps 1-4 produce `["tool", "skill"]` as above.
   - Session-mode narrowing intersects with `["tool"]`.
   - **Final grant: `["tool"]`.**

3. Change the operator config to grant `ext-X`: `["tool", "skill", "mcp_server", "automation.job"]` (fully permissive).
   **Expected:**
   - Source-tier ceiling still removes `automation.job`.
   - **Final grant in normal session: `["tool", "skill", "mcp_server"]`.**
   - Operator config cannot elevate beyond the source-tier ceiling.

4. Change the extension manifest to request only `["tool"]`.
   **Expected:**
   - Even though operator config and source tier allow more, the manifest request narrows to `["tool"]`.
   - **Final grant: `["tool"]`.**

5. Verify the grant computation returns not just the kind list but also per-kind permissions (e.g., read-only vs read-write) if applicable.
   **Expected:** Each granted kind includes the appropriate permission level, also computed as an intersection across the layers.

## Edge Cases

- All five layers allow everything: the grant is the full set of kinds the extension requested.
- Any single layer is empty: the final grant is empty (extension gets no access).
- Operator config is absent (not configured): it defaults to "no restriction from this layer," and the intersection proceeds with the remaining four layers.
- An extension with `source-tier="builtin"` has a higher ceiling than `community`, reflecting trust differentiation.
- Grant computation is deterministic: computing the same grant multiple times with the same inputs always produces the same output.
- The grant computation logs or returns metadata about which layer was the most restrictive, aiding debugging.
