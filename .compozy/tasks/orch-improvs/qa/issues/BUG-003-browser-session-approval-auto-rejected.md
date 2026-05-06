# BUG-003: Browser Session Approval Fixture Auto-Rejected Writes

**Severity:** Medium  
**Priority:** P1  
**Type:** Functional  
**Status:** Fixed

## Environment

- **Build:** local dev build from task_32 QA execution
- **OS:** macOS, isolated AGH lab from `qa/bootstrap-manifest.json`
- **Browser:** Playwright daemon-served web E2E
- **URL:** web session onboarding approval flow
- **Live provider/LLM:** acpmock fixture; native provider not required for this regression

## Summary

The browser session onboarding E2E clicked "Allow Once" but the runtime returned `409 pending permission not found` because the fixture auto-rejected write permission instead of presenting an interactive pending approval.

## Behavioral Impact

- **Operator/User Goal:** The operator cannot approve a pending permission in the onboarding flow.
- **Agent Behavior:** The mock session emits a rejected permission event before the UI can approve it.
- **Business Outcome:** Web permission UX appears broken even though the approval endpoint is valid.
- **Cross-Surface State:** UI action and runtime permission state disagreed.

## Reproduction

```bash
cd web
AGH_WEB_API_PROXY_TARGET=http://127.0.0.1:63022 \
  bun run test:e2e:daemon-served:raw e2e/session-onboarding.spec.ts
```

Observed before the fix:

- `POST /api/sessions/:id/approve` returned 409 with `pending permission not found`.

## Expected

The fixture must generate a pending write approval so the web flow can exercise the real approval endpoint and observe the accepted decision.

## Root Cause

`internal/testutil/acpmock/testdata/browser_session_lifecycle_fixture.json` used `permissions: "deny-all"`, which auto-rejected write permissions and never registered a pending approval request.

## Fix

The fixture now uses `permissions: "approve-reads"`, preserving automatic read approval while requiring real operator approval for writes.

## Verification

- `AGH_WEB_API_PROXY_TARGET=http://127.0.0.1:63022 bun run test:e2e:daemon-served:raw e2e/session-onboarding.spec.ts`
- Focused web set: `e2e/automation.spec.ts e2e/bridges.spec.ts e2e/session-onboarding.spec.ts`
- Full web E2E: `make test-e2e-web`

## Impact

- **Users Affected:** Web onboarding QA and permission regression coverage.
- **Frequency:** Always in the affected fixture path.
- **Workaround:** None.

## Related

- Test Case: TC-UI-001

