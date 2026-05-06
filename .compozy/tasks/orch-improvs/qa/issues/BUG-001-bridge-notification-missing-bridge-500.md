# BUG-001: Missing Bridge Subscription Returned Internal Error

**Severity:** High  
**Priority:** P1  
**Type:** Functional  
**Status:** Fixed

## Environment

- **Build:** local dev build from task_32 QA execution
- **OS:** macOS, isolated AGH lab from `qa/bootstrap-manifest.json`
- **Browser:** not applicable
- **URL:** `POST /api/tasks/{id}/notifications/bridges`
- **Live provider/LLM:** not required; reachable HTTP/UDS/CLI boundary

## Summary

Creating a task bridge notification subscription with a non-existent bridge instance returned a raw SQLite foreign-key/internal error instead of a deterministic domain error.

## Behavioral Impact

- **Operator/User Goal:** The operator cannot distinguish a missing bridge from daemon failure.
- **Agent Behavior:** Agent automation receives an opaque failure instead of a repairable "bridge not found" response.
- **Business Outcome:** Notification setup appears unstable and blocks deterministic remediation.
- **Cross-Surface State:** HTTP returned the wrong status; CLI/UDS needed the same domain classification.

## Reproduction

```bash
curl -i -X POST "$AGH_WEB_API_PROXY_TARGET/api/tasks/orch-qa-task-20260505/notifications/bridges" \
  -H 'content-type: application/json' \
  --data '{"bridge_instance_id":"bridge-missing","route_id":"route-1","target":{"peer_id":"peer-1","mode":"reply"}}'
```

Observed before the fix:

- HTTP returned a 500-style internal persistence error for a missing bridge instance.

## Expected

The API must validate bridge existence before persistence and return a deterministic not-found error without writing a subscription.

## Root Cause

`CreateTaskBridgeNotificationSubscription` wrote the subscription directly to the bridge task subscription store. The handler relied on the database foreign key instead of validating the bridge instance through the bridge service authority first.

## Fix

`internal/api/core/bridges.go` now calls `bridges.GetInstance` before `PutBridgeTaskSubscription` and maps bridge errors through the existing bridge error classifier. `internal/api/core/tasks_test.go` covers the missing-bridge 404 and verifies no persistence occurs.

## Verification

- `go test ./internal/api/core -run 'TestBaseHandlersTaskBridgeNotificationSubscription' -count=1`
- `go test -race ./internal/api/core -run 'TestBaseHandlersTaskBridgeNotificationSubscription' -count=1`
- Live HTTP evidence: `qa/evidence/runtime/09-http-notification-subscribe-missing-bridge-after-fix.txt`
- Live CLI evidence: `qa/evidence/runtime/09-cli-notification-subscribe-missing-bridge-after-fix.json`

## Impact

- **Users Affected:** Operators and agents configuring bridge notifications.
- **Frequency:** Always when a missing bridge id was submitted.
- **Workaround:** None.

## Related

- Test Case: TC-INT-003

