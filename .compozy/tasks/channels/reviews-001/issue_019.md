---
status: resolved
file: internal/channels/types.go
line: 445
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLo,comment:PRRC_kwDOR5y4QM623eJB
---

# Issue 019: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`InboundMessageEnvelope.normalize()` mutates the caller’s attachment slice.**

`normalized := e` only copies the slice header. Reassigning `normalized.Attachments[idx]` rewrites the same backing array held by `e.Attachments`, so a validation/normalization pass unexpectedly modifies the original envelope.

<details>
<summary>Possible fix</summary>

```diff
 func (e InboundMessageEnvelope) normalize() InboundMessageEnvelope {
 	normalized := e
+	normalized.Attachments = append([]MessageAttachment(nil), e.Attachments...)
 	normalized.ChannelInstanceID = strings.TrimSpace(normalized.ChannelInstanceID)
 	normalized.Scope = normalized.Scope.Normalize()
 	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
 	normalized.PeerID = strings.TrimSpace(normalized.PeerID)
 	normalized.ThreadID = strings.TrimSpace(normalized.ThreadID)
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/channels/types.go` around lines 439 - 445,
InboundMessageEnvelope.normalize() currently does normalized := e which only
copies the slice header so looping and assigning normalized.Attachments[idx]
mutates the original e.Attachments; fix by making a deep copy of the attachments
slice before normalizing (e.g., create a new slice with make(len(e.Attachments))
or use append to copy elements into a new backing array, then assign sanitized
MessageAttachment values into that new slice) so that normalize() updates
normalized.Attachments without modifying the caller's envelope; update the code
paths in InboundMessageEnvelope.normalize, the normalized variable handling, and
any use of normalized.Attachments to use the copied slice.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `InboundMessageEnvelope.normalize()` copies the attachments slice header only, then rewrites elements in place, which mutates the caller-owned backing array during normalization/validation.
  - I will deep-copy the attachments slice before normalizing it and add a regression test that proves the original envelope remains unchanged.
  - Resolution: `InboundMessageEnvelope.normalize()` now clones attachments in [internal/channels/types.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/channels/types.go:445), with regression coverage in [internal/channels/types_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/channels/types_test.go:263); verified with `make verify`.
