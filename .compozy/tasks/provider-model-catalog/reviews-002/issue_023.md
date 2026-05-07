---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/modelcatalog/sources.go
line: 160
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:e9a2aae4a7aa
review_hash: e9a2aae4a7aa
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 023: Deep-copy the curated model data here.
## Review Comment

`maps.Copy` only duplicates the map header. `ProviderConfig.Models.Curated` still shares the caller's backing array, so post-construction config edits can bleed into `ListModels()` and make this source stop behaving like a stable snapshot.

Based on learnings: Keep execution paths deterministic and observable in Go backend.

## Triage

- Decision: `invalid`
- Notes:
  - The current implementation already deep-copies provider model config state at construction time.
  - `cloneProviderModelConfigs` clones the curated slice, pointer fields, and `ReasoningEfforts`, so post-construction caller mutations do not alias into the source snapshot.
  - No code change is needed; this finding is stale against the current file.
