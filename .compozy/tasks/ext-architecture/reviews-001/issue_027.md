---
status: resolved
file: internal/extension/manifest.go
line: 547
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAan,comment:PRRC_kwDOR5y4QM62zlsz
---

# Issue 027: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Filter blank entries during normalization.**

This keeps whitespace-only items as `""`. For `resources.skills` / `resources.agents`, that later resolves to the extension root and can accidentally load every markdown file under the package on malformed input. Either drop blank entries here or reject them during validation.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/manifest.go` around lines 538 - 547, normalizeStrings
currently trims whitespace but preserves empty strings, which can later resolve
to the extension root (e.g., resources.skills/resources.agents) and cause
accidental loading; update normalizeStrings to filter out entries where
strings.TrimSpace(value) == "" (skip appending blank results) so the returned
slice contains only non-empty, trimmed strings. Locate the normalizeStrings
function and implement the check that ignores whitespace-only values before
append, ensuring callers no longer receive "" entries.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `normalizeStrings` trims whitespace but preserves empty results. For manifest resource lists, an empty string can later resolve to the extension root and widen loading far beyond the intended file or directory.
  Fix approach: filter whitespace-only entries during normalization and add a manifest-level regression test.
