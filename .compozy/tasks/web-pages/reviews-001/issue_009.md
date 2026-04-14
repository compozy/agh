---
status: resolved
file: internal/daemon/bridges.go
line: 190
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg37,comment:PRRC_kwDOR5y4QM63ZMHo
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don’t let one bad extension take down provider discovery.**

`ListProviders` aborts on the first unloadable or malformed bridge extension. That means one broken provider turns the whole bridges-provider endpoint into a 500 and hides otherwise healthy providers from the UI. Prefer logging and skipping invalid entries here so discovery stays usable.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/bridges.go` around lines 175 - 190, The current provider
discovery code in ListProviders returns an error on the first bad extension
(loadExtensionSnapshot, ext.Manifest nil, or missing
ext.Manifest.Bridge.Platform / Bridge.DisplayName), which aborts discovery;
change those early returns to log the specific failure via r.logger (including
info.Name and the error or missing field) and continue to the next extension so
one malformed or unloadable provider is skipped instead of failing the whole
ListProviders call; ensure the rest of the provider-building logic still appends
only valid providers to the results slice.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: provider discovery aborts on the first unloadable or malformed bridge-capable extension, which makes one bad extension hide every healthy provider from the UI.
- Fix approach: log malformed/unloadable entries and continue building the provider list from the remaining valid extensions.
