---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/bridges/resource_test.go
line: 379
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:681a35be945e
review_hash: 681a35be945e
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 021: Convert this flat case into a Should ... subtest and assert the exact rejection.
## Review Comment

Right now this only checks `err != nil`, so it will still pass if `ApplyResourceState` fails for the wrong reason. Please wrap it in a named subtest and check the expected error text/type, ideally with a sibling case for the untyped `nil` interface as well.

As per coding guidelines, "Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures" and "MUST have specific error assertions (ErrorContains, ErrorAs)".

## Triage

- Decision: `valid`
- Root cause: `TestBridgeResourceApplyRejectsTypedNilPlanWithoutReplacingInstances` only checks that `ApplyResourceState` returns some error. It does not lock down the specific typed-nil rejection path or verify sibling untyped-nil interface behavior.
- Fix plan: convert this coverage into `Should ...` subtests, assert the exact rejection text, and add the sibling untyped-nil interface case while preserving the no-replacement assertions.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
