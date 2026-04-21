## TC-INT-011: No-catalog peers remain joinable and discovery-empty in deterministic ways

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/config`, `internal/session`, `internal/network`
**Traceability:** Tasks 01-04; TechSpec no-catalog projection rules; shared-memory note about nil vs empty behavior.
**Execution Surfaces:** Loader, join payload, `greet`, explicit rich `whois`.
**Durable Regression Anchors:** `TestLoadAgentCapabilitiesMissingCatalogIsOptional`, `TestLoadWorkspaceAgentDefsLoadsAgentsWithoutCapabilityCatalog`, `TestManagerIntegrationCapabilityAwareJoinKeepsMissingCatalogProjectionEmpty`, `TestRouterWhoisRichCapabilityDiscoveryReturnsEmptyCatalogForUnknownIDsOrMissingCatalog`

### Objective

Verify agents with no local capability catalog still load and join successfully while emitting deterministic empty discovery shapes instead of `nil` or inconsistent omissions.

### Preconditions

- [ ] Temporary agent directory exists with `AGENT.md` and no capability sidecar/catalog.
- [ ] A channel-enabled session or equivalent network join path is available.

### Test Steps

1. Load an agent directory that has no capability catalog.
   - **Expected:** The agent loads successfully and the runtime treats capabilities as absent, not as a fatal error.
2. Join the peer to a network channel.
   - **Expected:** The join payload contains an empty capability slice, `peer_card.capabilities = []`, and no `agh.capabilities_brief` key.
3. Send an explicit rich `whois` request for the same peer.
   - **Expected:** The response returns `agh.capability_catalog.capabilities = []`.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Catalog | absent | Primary control condition |
| Rich request | `agh.include=["capability_catalog"]` | Explicit empty-catalog proof |

### Post-conditions

- Test peers can be removed from the channel.
- Evidence includes load, join, and explicit rich discovery artifacts for the same no-catalog peer.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Workspace discovery | No-catalog agent found via workspace scan | Agent still loads successfully |
| API payload | Inspect peer detail after join | API shows empty capabilities and omitted brief ext key |

### Related Test Cases

- `TC-INT-001`
- `TC-INT-008`
- `TC-INT-009`
- `TC-INT-010`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- This case stays in smoke because empty-state regressions often surface as brittle `nil` handling later in the pipeline.
