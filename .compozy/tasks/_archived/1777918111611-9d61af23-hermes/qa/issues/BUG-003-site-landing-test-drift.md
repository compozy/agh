# BUG-003: Site landing tests drifted from current landing copy

**Severity:** Medium
**Priority:** P1
**Type:** UI
**Status:** Fixed

## Environment

- **Build:** local Task 11 workspace
- **OS:** macOS
- **Browser:** Not applicable
- **URL:** `packages/site` test suite

## Summary

The site validation lane failed because landing page tests asserted outdated copy and card count for the current landing implementation.

## Reproduction

```bash
bun run --cwd packages/site test
```

Observed before the fix:

- `BentoSection` expected heading `Every step. Always replayable.`, while the component exposes `Every step Traceable.`
- `BentoSection` expected body copy that no longer exists in the current image-led bento card layout.
- `ExtensibilitySection` expected five cards including `Memory`, while the section title and feature list intentionally cover four cards: hooks, skills, automation, and extensions.

## Expected

The site source tests should validate the current public landing surface and fail only on real UI/content regressions.

## Root cause

The landing tests were stale after the landing content was reshaped. The component behavior was internally consistent: memory remains represented in the runtime/features sections, and the extensibility section contains four cards matching its heading.

## Fix

Updated `packages/site/components/landing/__tests__/landing.test.tsx` to assert the current bento grid/card count, accessible headings, exported images, and the four-card extensibility section.

## Verification

- `bun run --cwd packages/site test`
- Broader site typecheck/build and final repository gates are rerun in Task 11 after this fix.

## Impact

- **Users Affected:** Contributors running the site validation lane
- **Frequency:** Always
- **Workaround:** none

## Related

- Test Case: TC-REG-002
