---
status: resolved
file: internal/cli/extension_marketplace.go
line: 385
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562WUQ,comment:PRRC_kwDOR5y4QM63madS
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Close sources that are dropped by `sourceFilter`.**

`RegistrySource` has a `Close()` contract, but only the filtered slice is later wrapped by `MultiRegistry` and closed. Any backend rejected by `filterExtensionRegistrySources()` leaks its cleanup path on both success and error returns.  



Also applies to: 388-404

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/extension_marketplace.go` around lines 363 - 385,
configuredExtensionRegistrySources currently loads all RegistrySource from
loader(runtime) but only wraps the filtered slice in a MultiRegistry, leaking
unselected sources. After loader(runtime) returns, ensure you call Close() on
any source that is not included in filtered (i.e., removed by
filterExtensionRegistrySources) and also call Close() on all loaded sources
before returning on error paths. Update the function so that after computing
filtered := filterExtensionRegistrySources(sources, sourceFilter) you iterate
sources and Close() those not in filtered, and on any early return (err or
len(sources)==0 or len(filtered)==0) close all loaded sources before returning
(preserving/combining the original error as needed); reference
functions/variables configuredExtensionRegistrySources,
loader/defaultExtensionRegistrySourceLoader, filterExtensionRegistrySources,
RegistrySource.Close, and MultiRegistry to locate the code to change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `configuredExtensionRegistrySources()` returns only the filtered slice, so any loaded source excluded by `sourceFilter` is never wrapped by `MultiRegistry` and therefore never closed. The same leak exists on early returns after `loader(runtime)` succeeds. I will add explicit close handling for dropped and early-returned sources in `internal/cli/extension_marketplace.go` and add regression coverage in `internal/cli/extension_marketplace_test.go` even though that test file is outside the listed scope, because no in-scope test file exercises this helper path.
- Resolution: Added explicit close handling for filtered-out and early-returned extension registry sources in `internal/cli/extension_marketplace.go`, with regression coverage in `internal/cli/extension_marketplace_test.go`.
- Verification: `go test ./internal/cli`; `make verify`
