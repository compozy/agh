---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: pending
file: internal/modelcatalog/sources.go
line: 166
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6tZ,comment:PRRC_kwDOR5y4QM6-6btQ
---

# Issue 024: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Snapshot the provider configs deeply here.**

`maps.Copy` only clones the map header. Nested state inside `aghconfig.ProviderConfig`—notably the model metadata slices—remains aliased to the caller, so later config mutation can change this source's output after construction and can race with `ListModels`. Deep-copy the stored provider configs so the source is an immutable snapshot.

 

Based on learnings, "Keep execution paths deterministic and observable."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/modelcatalog/sources.go` around lines 160 - 166, The current
cloneConfigProviders function uses maps.Copy which only shallow-copies the map
header and leaves nested state in aghconfig.ProviderConfig (e.g., model metadata
slices) aliased; change cloneConfigProviders to produce a deep snapshot by
iterating over src and for each entry (key, cfg) copy the ProviderConfig value
into a new variable, deep-copy any slice or map fields (notably the model
metadata slices) into new slices/maps and assign the copied struct to
cloned[key], ensuring no shared references remain so subsequent mutations to the
original configs cannot affect the cloned snapshot used by ListModels.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
