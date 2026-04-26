---
status: resolved
file: internal/config/capabilities.go
line: 229
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4141178396,nitpick_hash:6eebfa2b895c
review_hash: 6eebfa2b895c
source_review_id: "4141178396"
source_review_submitted_at: "2026-04-20T15:11:23Z"
---

# Issue 002: Symlinks are silently skipped.
## Review Comment

`info.Mode().IsRegular()` returns `false` for symlinks, so capability files that are symlinks will be silently ignored. If this is intentional (for security or simplicity), consider documenting this behavior. If symlinks should be followed, use `os.Stat` on the path instead of `entry.Info()`.

## Triage

- Decision: `invalid`
- Root cause analysis: directory-mode loading intentionally accepts only regular files. `loadCapabilityCatalogDirectory(...)` checks `info.Mode().IsRegular()` before selecting `*.toml` or `*.json` entries.
- Why this is invalid: that behavior is already documented in [docs/rfcs/005_capability-catalogs-agent-directories.md](/Users/pedronauck/Dev/compozy/agh/docs/rfcs/005_capability-catalogs-agent-directories.md#L145) as "Only regular files with the selected extension are loaded," and the existing regression test `TestLoadAgentCapabilitiesDirectoryModeLoadsSelectedRegularFilesOnly` codifies the same contract.
- Additional reasoning: following symlinks here would broaden the filesystem trust boundary for agent catalogs rather than fixing a documented bug. The current behavior is a deliberate fail-closed loader rule, not an accidental omission.

## Resolution

- No code change required. The current `regular files only` contract remains the intended behavior for directory-mode capability catalogs.
- Verification:
  - `make verify`
