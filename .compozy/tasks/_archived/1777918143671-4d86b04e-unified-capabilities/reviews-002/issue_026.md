---
status: resolved
file: internal/session/manager_integration_test.go
line: 164
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:c65d28ef27f1
review_hash: c65d28ef27f1
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 026: Compute the expected digest instead of reading it from the fixture.
## Review Comment

`capabilityAgent.Capabilities.Capabilities[0].Digest` starts empty in this literal, so this assertion only verifies digest projection if some earlier helper mutated the catalog in place. Deriving the expected value with `aghconfig.CanonicalCapabilityDigest(...)` makes the new check deterministic.

## Triage

- Decision: `valid`
- Notes:
  The expected digest is currently taken from `capabilityAgent.Capabilities...Digest`, but that field is populated as a side effect of `AgentDef.Validate()`. The assertion is more robust if it derives the digest explicitly from the capability definition being projected.
  I will compute the expected digest with `aghconfig.CanonicalCapabilityDigest(...)` and compare against that deterministic value.
  Fixed and verified with targeted package tests plus `make verify`.
