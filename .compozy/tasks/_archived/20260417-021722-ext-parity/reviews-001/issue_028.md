---
status: resolved
file: internal/bundles/resource_test.go
line: 71
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:17d9e9346374
review_hash: 17d9e9346374
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 028: Make the wrong-plan failure assertion specific.
## Review Comment

`err != nil` will pass for any unrelated failure inside `Apply`, so this does not prove the type-check branch is exercised. Please assert the expected error type or at least an identifying substring.

As per coding guidelines, "`**/*_test.go`: MUST have specific error assertions (ErrorContains, ErrorAs)`."

## Triage

- Decision: `VALID`
- Notes: `TestBundleActivationBuildComposesTypedBundleDependency` still treats any non-nil error from `Apply(nonBundleActivationPlan{})` as success, which would hide unrelated failures inside `Apply`. The test should assert the identifying wrong-plan-type message so it pins the actual guard being exercised.
