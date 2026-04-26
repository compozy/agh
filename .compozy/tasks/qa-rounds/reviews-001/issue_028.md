---
status: resolved
file: web/src/storybook/web-storybook-stories-and-fixtures.test.tsx
line: 23
severity: minor
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:dc902e7f90ef
review_hash: dc902e7f90ef
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 028: Route system imports through a public barrel instead of deep internals.
## Review Comment

Line 23 imports `@/systems/network/components/stories/network-workspace-shell.stories` directly from a system-internal path. Please expose a public test-facing entrypoint (or import via an allowed public barrel) so this test does not depend on network internals.

As per coding guidelines, "Cross-system imports MUST only go through the public barrel (`@/systems/<domain>`). Never reach into another system's internals".

## Triage

- Decision: `valid`
- Notes: The Storybook regression test imports the network workspace shell story through `@/systems/network/components/stories/...`, which bypasses the system boundary. There is no existing public test-facing entrypoint for this story, so the minimal required out-of-scope code addition is `web/src/systems/network/storybook.ts`, a network-owned Storybook barrel that the regression test can import via `@/systems/network/storybook`.
