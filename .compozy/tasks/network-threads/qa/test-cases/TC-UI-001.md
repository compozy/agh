## TC-UI-001: Web Network Thread And Direct Navigation - Operator Can Act On Real State

**Priority:** P1
**Type:** UI/Visual
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05
**Execution Class:** Browser E2E

---

### Objective

Verify that the Web `/network` surface presents actual thread/direct/work state, uses final route shapes, and captures browser artifacts with `network_selected_thread` and `network_selected_direct` instead of legacy peer-room state.

### Preconditions

- [ ] Web dev server is started with `AGH_WEB_API_PROXY_TARGET` from the bootstrap manifest when applicable.
- [ ] Browser-use is available, or `agent-browser` fallback is documented.
- [ ] Scenario state includes at least one public thread and one direct room.
- [ ] Authentication/session state is available if the app requires it.

### Test Steps

1. **Open public thread list**
   - Input: Navigate to `/network/builders/threads`.
   - **Expected:** Thread list renders real `builders` thread summaries and no direct-room transcript details.

2. **Open public thread detail**
   - Input: Navigate to `/network/builders/threads/thread_launch_review`.
   - **Expected:** Timeline shows public messages, composer targets `surface:"thread"`, and screenshot evidence is captured.

3. **Open direct-room list**
   - Input: Navigate to `/network/builders/directs`.
   - **Expected:** Direct rooms render with server-provided two-peer membership.

4. **Open direct-room detail**
   - Input: Navigate to `/network/builders/directs/$DIRECT_ID`.
   - **Expected:** Timeline shows only direct-room messages, composer targets `surface:"direct"`, and no public-thread messages leak.

5. **Capture browser artifact state**
   - Input: Use the repo/browser artifact capture path from `web/e2e`.
   - **Expected:** Artifact contains `network_selected_thread` on thread detail, `network_selected_direct` on direct detail, and no active `network_selected_peer` except intentional negative assertions.

6. **Invalid route probe**
   - Input: Navigate to missing thread/direct IDs.
   - **Expected:** UI shows operator-readable error or empty state without invented controls or misleading metrics.

### Behavioral Evidence

- Operator journey: inspect and act on real thread/direct state through Web.
- Live agent/LLM behavior: not required for this UI case, but scenario data should come from the real daemon or QA harness.
- Artifacts produced and used: screenshots and browser route artifacts under `qa/screenshots/`.
- Cross-surface assertions: Web visible IDs match CLI/API values from P0 scenarios.

### Disruption Probes

- Missing thread route shows operator-readable error state.
- Missing direct-room route shows operator-readable error state.
- Browser artifact capture never falls back to legacy `network_selected_peer` for active route state.

### Related Test Cases

- TC-SCEN-001
- TC-SCEN-002
- TC-SCEN-003
