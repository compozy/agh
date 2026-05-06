---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/situation/service_test.go
line: 96
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4233550358,nitpick_hash:87ae2e7dce97
review_hash: 87ae2e7dce97
source_review_id: "4233550358"
source_review_submitted_at: "2026-05-06T05:52:14Z"
---

# Issue 011: Make the channel-source precedence explicit in this fixture.
## Review Comment

These changes now feed three different channel identifiers into the same scenario (`CoordinationChannelID`, `NetworkChannel`, and metadata `coordination_channel_id`), but the test never pins which one `ContextForSession` is supposed to surface. That means a regression can read the wrong source and still pass. Either collapse these to one canonical value here or add an explicit assertion on `payload.CoordinationChannel.Channel.ID`.

As per coding guidelines, "Ensure tests verify behavior outcomes, not just function calls."

Also applies to: 143-145, 150-168

## Triage

- Decision: `VALID`
- Root cause: the fixture feeds three different channel sources but only asserts workflow metadata, so the test would still pass if `ContextForSession` chose the wrong channel precedence.
- Fix approach: Add an explicit assertion for the surfaced coordination-channel ID so the precedence is pinned by behavior.
