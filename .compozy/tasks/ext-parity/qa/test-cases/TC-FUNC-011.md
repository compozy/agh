# TC-FUNC-011: Surface registry rejects illegal kind for extension

**Priority:** P0
**Type:** Functional
**Package:** internal/extension/surfaces
**Related Tasks:** 04

## Objective

Validate that the surface registry enforces kind restrictions per surface type. When an extension attempts to publish or request access to a resource kind that is reserved for daemon-internal use (e.g., `bundle.activation` or other daemon-only kinds), the surface registry must deny the request before the extension handshake completes. This is the first line of defense in the grant computation pipeline.

## Preconditions

- The surface registry is initialized with at least two surface definitions:
  - `agent` surface: allows kinds `["tool", "skill", "mcp_server"]`.
  - `daemon` surface: allows all 10 kinds including daemon-only kinds like `bundle.activation` and `bridge.instance`.
- An extension manifest declares `surface="agent"` and requests kinds `["tool", "bundle.activation"]`.

## Test Steps

1. Register the `agent` surface with its allowed kinds set `["tool", "skill", "mcp_server"]`.
   **Expected:** Registration succeeds. The surface is queryable.

2. Register the `daemon` surface with the full kind set.
   **Expected:** Registration succeeds.

3. Submit an extension handshake with `surface="agent"` and manifest requesting `kinds=["tool"]`.
   **Expected:** The handshake proceeds. The `tool` kind is legal for the `agent` surface.

4. Submit an extension handshake with `surface="agent"` and manifest requesting `kinds=["tool", "bundle.activation"]`.
   **Expected:** The handshake is rejected before completion. The error clearly identifies `bundle.activation` as an illegal kind for the `agent` surface. No partial grant is issued (the entire handshake fails, not just the illegal kind).

5. Submit an extension handshake with `surface="daemon"` and manifest requesting `kinds=["bundle.activation"]`.
   **Expected:** The handshake proceeds (assuming the caller has daemon-level authority). The `bundle.activation` kind is legal for the `daemon` surface.

6. Submit an extension handshake with `surface="unknown-surface"`.
   **Expected:** The handshake is rejected with an error indicating the surface does not exist.

## Edge Cases

- An extension requests an empty kinds list (`[]`): the handshake either succeeds with zero grants or is rejected as meaningless, depending on policy.
- An extension requests a kind that does not exist in any surface (e.g., `"nonexistent.kind"`): rejected as unknown kind, distinct from the "illegal for this surface" error.
- Surface definitions are immutable after registration: attempting to modify a surface's allowed kinds after startup is rejected.
- Case sensitivity: `"Tool"` vs `"tool"` -- the registry enforces exact case matching and rejects mismatched casing.
- An extension requests the same kind twice in its manifest (e.g., `["tool", "tool"]`): deduplicated or rejected, but does not result in double grants.
