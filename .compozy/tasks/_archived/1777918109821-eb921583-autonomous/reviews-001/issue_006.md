---
status: resolved
file: internal/api/core/agent_channels.go
line: 743
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:23a94fa40bef
review_hash: 23a94fa40bef
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 006: coordinationMetadataFromEnvelope: Fallback to entire Ext as metadata is fragile.
## Review Comment

Lines 754-762 attempt to unmarshal the entire `envelope.Ext` map as coordination metadata if no known key is found. This could incorrectly succeed if `Ext` happens to contain fields matching `CoordinationMessageMetadataPayload`, even when it's not actual coordination metadata.

Consider whether this fallback is intentional or if it should be removed to avoid false positives.

---

## Triage

- Decision: `VALID`
- Notes: `coordinationMetadataFromEnvelope` accepts known extension keys first, then marshals the entire extension map and may treat arbitrary extension fields as coordination metadata if their names happen to match. That can convert unrelated network extension data into agent-channel messages. Fix by removing the whole-map fallback and only accepting explicit coordination metadata keys; add handler coverage that ignores misleading extension maps.
