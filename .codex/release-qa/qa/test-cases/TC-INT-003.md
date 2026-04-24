## TC-INT-003: Whois And Capability Exchange

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-24
**Last Updated:** 2026-04-24

### Objective

Verify AGH Network peer discovery and capability exchange, the primary coordination primitive for an Agent OS.

### Preconditions

- Network is enabled.
- At least two sessions expose network capability catalogs.

### Test Steps

1. Join sessions with distinct capability catalogs.
   **Expected:** Peer cards expose capability briefs without mutating original catalog data.

2. Send a `whois` request for a capability.
   **Expected:** Matching peers respond with `whois` responses and capability catalog metadata.

3. Verify persisted audit/timeline data.
   **Expected:** Request and response messages are queryable through network APIs/UI surfaces.

### Edge Cases & Variations

| Variation           | Input           | Expected Result                         |
| ------------------- | --------------- | --------------------------------------- |
| No capability match | Unknown query   | No false positive responder is emitted. |
| Directed whois      | `--to` one peer | Only the target peer responds.          |
