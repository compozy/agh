## TC-UI-001: Web peer-detail UX and typed-client alignment

**Priority:** P1
**Type:** UI
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-20
**Last Updated:** 2026-04-20

---

### Objective

Verify that the `web/` network surface consumes the unified backend capability contract and renders peer capability data clearly in the operator UI without recipe-era terminology, missing fields, or reliance on raw API extension blobs.

---

### Preconditions

- [ ] The backend/API surface from `TC-INT-004` is available or mocked consistently.
- [ ] The executor can open the network route in the browser or run the existing route/component regression tests.
- [ ] Web verification commands are available: `make web-lint`, `make web-typecheck`, and relevant web tests.

---

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Peer detail fixture | Brief capability summaries plus rich `capability_catalog` | Used for the positive path |
| Brief-only peer | Peer with summaries but no rich catalog | Used to verify graceful degradation |
| Empty state | No peer selected | Used to verify idle fallback |
| Error state | Failing peer detail request | Used to verify error fallback |

---

### Test Steps

1. Open the main network route and switch to the Peers view.
   - **Expected:** The page loads without recipe terminology and shows a peer list driven by the typed backend contract.

2. Select a peer whose detail payload includes both `peer_card.capabilities` and `capability_catalog`.
   - **Expected:** The detail panel shows capability ID, summary, version, outcome, requirements, context, expected artifacts, constraints, execution outline, and examples when present.

3. Verify the detail panel behavior for a peer with brief summaries but no rich catalog.
   - **Expected:** The UI still renders the brief summary cleanly and does not crash or render recipe-era placeholders.

4. Exercise the empty, loading, and error states for peer detail.
   - **Expected:** Each fallback state is clear, capability-safe, and free of broken rendering or stale content.

5. Review the page at desktop, tablet, and mobile widths.
   - **Expected:** Capability rows remain readable and the peer-detail panel stays usable without clipping or hidden capability metadata.

6. Run the required web checks after the UI scenario is stable.
   - **Expected:** `make web-lint` and `make web-typecheck` pass, and the targeted network route/component tests agree with the operator-visible behavior.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Rich capability with optionals | Version, requirements, execution outline present | All rendered in the correct sections |
| Brief-only capability | No `capability_catalog` detail | Summary still renders cleanly |
| No capability summaries | Empty peer capability state | Panel remains stable with no recipe fallback |
| Responsive widths | 1280 / 768 / 375 | Layout remains readable and usable |

---

### Traceability

- Tasks: `task_06`, `task_04`
- TechSpec: `API Endpoints`, `Integration Points`, `Testing Approach`
- ADRs: `ADR-001`, `ADR-002`, `ADR-003`
- Primary surfaces: `web/src/systems/network/adapters/network-api.ts`, `web/src/hooks/routes/use-network-page.ts`, `web/src/systems/network/components/network-peer-detail-panel.tsx`

---

### Evidence to Capture

- At least one screenshot per viewport if the browser/manual path is used
- Output from `make web-lint` and `make web-typecheck`
- Targeted route/component test evidence if available

---

### Notes

- This case is intentionally operator-facing. Passing backend tests alone is not enough if the network UI still leaks the old model or hides the new typed fields.
