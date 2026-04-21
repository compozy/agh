---
status: resolved
file: web/src/systems/session/components/stories/copy-button.stories.tsx
line: 37
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:84101a68d6ca
review_hash: 84101a68d6ca
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 027: Stub only writeText method instead of replacing the entire navigator.clipboard object.
## Review Comment

Replacing the entire `navigator.clipboard` object via `Object.defineProperty(navigator, "clipboard", ...)` is not recommended because it can break third-party code and libraries expecting native Clipboard API behavior, including permission checks and secure context enforcement. Instead, directly override only the `writeText` method:

```javascript
const originalWriteText = navigator.clipboard.writeText;
navigator.clipboard.writeText = async () => undefined;

// ... test code ...

navigator.clipboard.writeText = originalWriteText;
```

This maintains compatibility and isolates the mock to only what the test requires.

## Triage

- Decision: `valid`
- Notes: The story harness replaces the entire `navigator.clipboard` object even though the component only depends on `writeText()`. That broad replacement is unnecessary and less faithful to browser behavior because it drops the rest of the Clipboard surface. Fix by overriding only `writeText` when a clipboard object exists, while keeping a minimal fallback path for environments where Storybook does not expose `navigator.clipboard`.

## Resolution

- Narrowed the story harness to create a clipboard object only when the environment lacks one, and otherwise override only `navigator.clipboard.writeText` while preserving the existing clipboard object.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.
