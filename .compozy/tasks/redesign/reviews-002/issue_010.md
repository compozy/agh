---
status: resolved
file: packages/ui/src/components/collapsible.test.tsx
line: 45
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:7f420352ce8a
review_hash: 7f420352ce8a
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 010: Strengthen keepMounted assertion with closed-state visibility check.
## Review Comment

Right now this test only proves DOM persistence. Add a “closed but not visible” expectation to catch regressions where content stays visually open.

## Triage

- Decision: `valid`
- Reasoning: The `keepMounted` test only proves the content node remains in the DOM. It does not verify that the closed panel is actually hidden, so a visually-open regression could slip through.
- Root cause: The assertion checks persistence without checking closed-state visibility semantics.
- Fix plan: Extend the test to confirm the content remains mounted while no longer being visible/open after closing.

## Resolution

- Tightened `packages/ui/src/components/collapsible.test.tsx` to assert both DOM persistence and closed-state invisibility after toggling shut.
- Verified with `make verify` after all batch changes.
