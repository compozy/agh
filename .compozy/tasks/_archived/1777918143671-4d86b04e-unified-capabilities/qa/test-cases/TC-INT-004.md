## TC-INT-004: Discovery, peer details, and typed API contract alignment

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-20
**Last Updated:** 2026-04-20

---

### Objective

Verify that brief discovery, rich discovery, peer details, and daemon API contracts all surface the same unified capability model and no longer expose recipe-specific payloads or require raw `ext` parsing on API-visible paths.

---

### Preconditions

- [ ] A peer with a populated capability catalog is available in a local test channel.
- [ ] The executor can exercise `greet`, `whois`, HTTP, and UDS network surfaces.
- [ ] The peer detail path can expose a known rich capability catalog.

---

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Peer A | Catalog-bearing peer | Used for brief and rich discovery |
| Whois include | `["capability_catalog"]` | Triggers rich discovery |
| Capability filter | One known ID and one missing ID | Used to verify filter semantics |
| API endpoints | Peer list and peer detail endpoints | HTTP and/or UDS surfaces |

---

### Test Steps

1. Observe the peer through the brief discovery path (`greet` / peer list).
   - **Expected:** The peer advertises capability summaries through typed `peer_card.capabilities` entries and `artifacts_supported` includes `"capability"`.

2. Request rich discovery with `agh.include = ["capability_catalog"]` and an optional `agh.capability_ids` filter.
   - **Expected:** The response includes a rich `capability_catalog` with `id`, `summary`, `outcome`, `digest`, and optional structured fields; filtering preserves order and returns `[]` for unknown-only requests.

3. Fetch the same peer through daemon HTTP and UDS peer detail surfaces.
   - **Expected:** Both API surfaces expose the same typed capability contract, including `peer_card.capabilities` as brief typed objects and `capability_catalog` as the rich typed catalog when known.

4. Inspect API-visible `ext` payloads on peer list/detail responses.
   - **Expected:** Raw `agh.capabilities_brief` and `agh.capability_catalog` are stripped from API-visible `ext`; clients do not need to parse those blobs.

5. Confirm there is no recipe-era vocabulary or field shape in the peer list, peer detail, or rich discovery output.
   - **Expected:** All surfaced payloads and labels use the unified capability model only.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Known rich catalog | Peer detail after explicit rich discovery | Typed `capability_catalog` present and coherent with brief discovery |
| Unknown filter | `agh.capability_ids` contains only missing IDs | `capability_catalog.capabilities = []` |
| Empty catalog peer | Peer with no local catalog | Brief discovery empty, transfer support still advertised |
| Partial rich update | Filtered rich response | Existing brief summaries stay coherent for unrelated capability IDs |

---

### Traceability

- Tasks: `task_04`
- TechSpec: `System Architecture`, `API Endpoints`, `Testing Approach`
- ADRs: `ADR-001`, `ADR-002`, `ADR-003`
- Primary surfaces: `internal/network/capability_brief.go`, `internal/network/capability_catalog.go`, `internal/api/contract/contract.go`, `internal/api/core/network*.go`

---

### Evidence to Capture

- Brief discovery payload sample
- Rich discovery payload sample with and without filter
- HTTP and/or UDS peer detail payload sample
- Evidence that raw capability ext blobs are absent from API-visible payloads

---

### Notes

- This case proves the backend contract that task_06 and the site docs rely on. If it fails, frontend and documentation evidence is not trustworthy.
