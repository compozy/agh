# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Align `web/` network UX with the unified capability contract from task_04: typed `capability_catalog`, brief `peer_card.capabilities`, and `kind:"capability"` replacing `recipe` on the wire.

## Important Decisions

- Expose typed frontend aliases (`NetworkCapability`, `NetworkCapabilityBrief`, `NetworkCapabilityCatalog`, `NetworkPeerCapabilityView`) derived from the generated OpenAPI types instead of hand-rolled payloads.
- Merge brief + catalog entries into one `NetworkPeerCapabilityView[]` sorted by id; the Capabilities section tone flips to `accent` when any row has rich detail and shows a `brief|detailed` mono-badge next to the count.
- Protocol kind labels switch from `recipe` to `capability` in `network-channel-detail-panel`, `design-system-showcase`, and the UI kit `kind-chip` story so the valid-kind registry matches task_02/04.

## Learnings

- Route-level mocks pass `capability_catalog: null` to cover the brief-only fallback; peer-detail asserts both the brief-only and detailed-with-catalog flows.
- `MonoBadge` only needs `tone="accent"` for the detailed indicator — no extra UI kit changes were required.

## Files / Surfaces

- web/src/systems/network/types.ts
- web/src/systems/network/lib/network-formatters.ts (+ new test)
- web/src/systems/network/components/network-peer-detail-panel.tsx (+ refreshed test)
- web/src/systems/network/components/network-channel-detail-panel.tsx (recipe → capability)
- web/src/systems/network/mocks/fixtures.ts (capability_catalog)
- web/src/systems/network/index.ts (re-exports for new types/helpers)
- web/src/routes/_app/-network.test.tsx (new unified-capability route case)
- web/src/components/design-system-showcase.tsx (KINDS list)
- packages/ui/src/components/kind-chip.tsx (+ stories)

## Errors / Corrections

## Ready for Next Run

- Web lint, typecheck, and the full 1517-test vitest suite pass after the changes. No compat shims were introduced; the frontend consumes `capability_catalog` and the brief `peer_card.capabilities` list directly.
