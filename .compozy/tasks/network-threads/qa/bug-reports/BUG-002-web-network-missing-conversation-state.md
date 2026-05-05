# BUG-002: Web network detail routes rendered composer controls before missing conversations resolved

## Status

Fixed.

## Severity / Priority

- Severity: Medium
- Priority: P1

## Originating Test Cases

- `TC-UI-001`

## Confirmed Failure

Entry routes:

```text
http://localhost:3001/network/builders/threads/thread_missing_qa
http://localhost:3001/network/builders/directs/direct_missing_qa
```

Initial evidence:

- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/browser/network-missing-thread-after-wait.snapshot.txt`
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/browser/network-missing-direct-after-wait.snapshot.txt`
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/api-missing-thread.stdout`
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/api-missing-direct.stdout`

Observed behavior:

- Missing thread route showed the existing-thread empty state and a `Reply...` textbox while the thread detail query was still unresolved.
- Missing direct route showed the direct composer placeholder `Message @peer...` while the direct detail query was still unresolved.
- The API correctly rejected the same resources (`404` for the missing thread, `400` for the invalid direct id), so the UI state was misleading.

## Root Cause

`DirectRoom` and `ThreadOverlay` only branched on final detail errors. During the detail query retry window, the detail object was absent and the error had not materialized yet, but both components still rendered their normal timeline/composer paths.

The query layer also used the default TanStack Query retry behavior for conversation details, delaying final 4xx errors even though missing or invalid conversation identifiers are not transient failures.

## Fix

- `web/src/systems/network/components/directs/direct-room.tsx`: render only a loading timeline while the direct detail is unresolved, and withhold the direct composer until the room detail exists.
- `web/src/systems/network/components/thread-overlay/thread-overlay.tsx`: render loading root/replies while the thread detail is unresolved, and withhold the reply composer until the thread detail exists.
- `web/src/systems/network/lib/query-options.ts`: do not retry 4xx thread/direct detail failures; keep a short retry budget for transient detail failures.
- `web/src/systems/network/components/directs/direct-room.test.tsx`: added coverage for unresolved direct detail and missing direct detail.
- `web/src/systems/network/components/thread-overlay/thread-overlay.test.tsx`: added coverage for unresolved thread detail and missing thread detail.
- `web/src/systems/network/lib/query-options.test.ts`: added coverage for 4xx no-retry and transient retry-budget behavior.

## Verification

Commands:

```bash
bunx vitest run web/src/systems/network/components/directs/direct-room.test.tsx web/src/systems/network/components/thread-overlay/thread-overlay.test.tsx web/src/systems/network/lib/query-options.test.ts
make web-lint
make web-typecheck
make test-e2e-web
```

Passing evidence:

- Targeted Vitest run: `3 passed`, `20 passed`.
- `make web-lint`: `Found 0 warnings and 0 errors`.
- `make web-typecheck`: PASS.
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/test-e2e-web-after-ui-fix.log`: `19 passed`.
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/browser/network-missing-thread.snapshot.txt`: contains `Thread unavailable` and no `Reply...` textbox.
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/browser/network-missing-direct.snapshot.txt`: contains `Direct room unavailable` and no `Message @peer...` textbox.
