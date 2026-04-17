---
status: resolved
file: web/playwright.config.ts
line: 25
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:e7e757442ec9
review_hash: e7e757442ec9
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 038: Consider enabling trace/screenshot on failure for CI debugging.
## Review Comment

With all artifacts disabled, diagnosing flaky or failing E2E tests in CI becomes difficult. Playwright supports conditional artifact capture.

## Triage

- Decision: `invalid`
- Notes:
  This Playwright lane already captures deterministic trace and screenshot
  artifacts through `BrowserArtifactSession` and persists them into the shared
  E2E artifact model. Enabling global Playwright `trace`/`screenshot` output in
  the config would duplicate artifacts, increase CI volume, and cut across the
  harness' deliberate artifact ownership without addressing a current failure.

## Resolution

- No code change. Existing browser artifact capture already provides failure
  traces and screenshots for this lane without duplicating Playwright output.
