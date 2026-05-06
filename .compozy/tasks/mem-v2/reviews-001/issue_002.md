---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/api/contract/settings.go
line: 350
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Isc,comment:PRRC_kwDOR5y4QM6-UFVc
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_

**Version or preserve the old `workspace` payload shape.**

Replacing `enabled/global_dir/dream` with `toml_path/auto_create` changes a public settings payload with no compatibility shim. Existing HTTP/UDS clients will deserialize a different `workspace` object and silently lose those fields. Please keep the legacy fields during the transition or move the new layout behind a versioned envelope.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/contract/settings.go` around lines 348 - 350, The
SettingsMemoryWorkspacePayload struct change removed legacy fields (enabled,
global_dir, dream) and will break clients; restore backward compatibility by
either (A) adding the old fields back into SettingsMemoryWorkspacePayload
(enabled bool, global_dir string, dream string) alongside the new TOMLPath and
AutoCreate so deserialization preserves both shapes, or (B) introduce a
versioned envelope struct (e.g., SettingsWorkspaceV1 and SettingsWorkspaceV2)
and update the API handler/serializer to emit a version tag and
marshal/unmarshal the appropriate struct; locate the
SettingsMemoryWorkspacePayload type and the related serializer/deserializer code
paths to implement one of these options.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - This repo’s root `CLAUDE.md` defines AGH as a greenfield alpha with explicit "Zero Legacy Tolerance" and forbids compatibility shims, aliases, dual fields, or migration bridges for removed public shapes.
  - The current `SettingsMemoryWorkspacePayload` and conversion code intentionally hard-cut to the new `toml_path` / `auto_create` shape; restoring `enabled`, `global_dir`, or `dream` just to preserve an older payload would violate the repository policy for this codebase.
  - No code change is required for this batch item.
