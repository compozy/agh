## TC-UI-010: Every Create Entry Point Opens the Provider-Aware Dialog and Submits the Chosen Provider

**Priority:** P0
**Type:** UI
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-21
**Last Updated:** 2026-04-21
**Module:** Web create flow
**Traceability:** Task 06 dialog-flow requirements; ADR-004; TechSpec web impact analysis and UI integration tests

---

### Objective

Verify that every web session-create entrypoint opens the dialog, prepopulates agent/workspace/default provider, exposes workspace-visible provider options, and submits the selected provider into the session create mutation.

---

### Preconditions

- [ ] Browser execution environment is available at desktop, tablet, and mobile viewports.
- [ ] Workspace fixture `WS-PROVIDER-MATRIX` is available to the web app.
- [ ] The chosen agent has a known default provider and at least one alternate provider.
- [ ] Network inspection or request capture is available in the browser.

---

### Test Steps

1. Trigger session creation from each supported web entrypoint, including the sidebar path.
   **Expected:** Each entrypoint opens the same dialog instead of creating a session immediately.

2. Inspect the initial dialog state.
   **Expected:** The chosen agent, active workspace, and default provider are prefilled, and the provider picker options match the workspace detail payload.

3. Change the provider selection from the default to an alternate provider and submit the dialog.
   **Expected:** The create mutation sends the selected provider.

4. Observe the resulting session view.
   **Expected:** The session loads successfully and the effective provider is visible in the client-visible session metadata, such as the provider badge or header.

5. Repeat the dialog open on tablet and mobile breakpoints.
   **Expected:** The dialog remains usable, the picker is visible, and the controls remain actionable.

---

### Evidence to Capture

- Screenshots of the opened dialog on desktop and at least one smaller breakpoint.
- Network capture of the create mutation showing the selected provider.
- Post-create UI evidence showing the effective provider in the session UI.
- Optional comparison with workspace detail payload from `TC-INT-009`.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Default provider accepted | No picker change | Submitted provider matches the default prefill. |
| Alternate provider chosen | Change picker selection | Submitted provider matches the chosen alternate. |
| Different entrypoints | Sidebar and route-level create entrypoint | Both paths open the same dialog and keep parity. |
| Smaller viewport | Tablet or mobile | Dialog remains usable and provider picker stays accessible. |

---

### Related Test Cases

- `TC-INT-009` for workspace provider catalog
- `TC-INT-008` for backend surface parity on the created session

---

### Notes

If any entrypoint still bypasses the dialog, treat it as a blocking regression even if the rest of the flow works.
