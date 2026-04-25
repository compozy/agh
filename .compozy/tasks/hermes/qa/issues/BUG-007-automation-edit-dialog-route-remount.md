# BUG-007: Automation edit dialog loses state when route motion key catches up

## Status

Fixed in Task 11.

## Severity

P0 web E2E regression. The automation Jobs page could open the edit dialog for a single frame and then immediately lose it because the app route shell remounted the Jobs route on the same user click.

## Reproduction

Run:

```bash
bun run --cwd web test:e2e:daemon-served:raw --grep "operator can inspect automation"
```

Then navigate Jobs -> Triggers -> Jobs, select the seeded job, and click `Edit`.

## Observed

The `automation-job-form` never became visible. Playwright trace evidence showed the dialog briefly mounted with opacity `0`, then the route shell key changed from `/` to `/jobs`, unmounting the Jobs route and dropping the local editor state.

## Expected

Opening a route-local dialog should not remount the current route. The app route motion key must use a reactive router location source that is already current for the rendered route.

## Root Cause

`web/src/routes/_app.tsx` keyed the motion shell with `router.latestLocation.pathname || locationPathname`. `latestLocation` is mutable and not a stable render signal. It could remain stale during navigation, then catch up during an unrelated local state update, causing `AnimatePresence` to replace the route and discard local UI state.

## Fix

Keyed the app route motion shell directly from `useLocation({ select: location => location.pathname })` and updated the route-shell test to cover stale `router.latestLocation` values.

## Verification Evidence

- Failing full web E2E: `.compozy/tasks/hermes/qa/logs/final/make-test-e2e-web.log`
- Failing focused repro: `.compozy/tasks/hermes/qa/logs/final/failure-repro/web-e2e-automation-focused.log`
- Route regression proof: `.compozy/tasks/hermes/qa/logs/final/failure-repro/web-route-motion-key-after-fix.log`
- Focused E2E after fix: `.compozy/tasks/hermes/qa/logs/final/failure-repro/web-e2e-automation-focused-after-fix.log`
- Full web E2E after fix: `.compozy/tasks/hermes/qa/logs/final/make-test-e2e-web-after-fix.log`
- Web lint/typecheck/test after fix:
  - `.compozy/tasks/hermes/qa/logs/final/make-web-lint-after-fix.log`
  - `.compozy/tasks/hermes/qa/logs/final/make-web-typecheck-after-fix.log`
  - `.compozy/tasks/hermes/qa/logs/final/make-web-test-after-fix.log`
