---
status: resolved
file: packages/ui/src/components/sidebar.test.tsx
line: 91
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:5007cfc40bfa
review_hash: 5007cfc40bfa
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 033: Re-querying nodes after rerender is a best practice but the stated concern is not applicable here.
## Review Comment

The rail and panel elements are never conditionally rendered or keyed in a way that causes remounting when the `collapsed` prop changes. Both elements persist in the DOM across rerenders—only their properties (width, aria-hidden) change. References captured before `rerender` remain valid.

That said, re-querying after `rerender` follows React Testing Library best practices of testing current DOM state rather than relying on captured references, which improves test resilience to implementation changes. However, this is a stylistic improvement, not a correctness issue.

## Triage

- Decision: `invalid`
- Notes:
  - The review itself states the claimed remount concern is not applicable here. The rail and panel nodes are not remounted across this rerender path, so the captured references remain valid.
  - Re-querying after rerender would be a stylistic test-hardening change, not a fix for an actual correctness issue in the current code.
