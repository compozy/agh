---
status: resolved
file: internal/extension/bundle.go
line: 169
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__Bz,comment:PRRC_kwDOR5y4QM63zbyT
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Normalize bundle/profile names before duplicate checks.**

`LoadBundleSpecs` and `BundleSpec.Validate` treat `Foo` and `foo` as different keys, but activation later resolves bundles and profiles with `strings.EqualFold`. That lets case-only duplicates load successfully and then makes activation resolution ambiguous.



Also applies to: 180-203

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/bundle.go` around lines 147 - 169, When loading bundle
specs in LoadBundleSpecs, normalize bundle and profile names to a consistent
case before performing duplicate checks and before calling BundleSpec.Validate:
after obtaining spec from loadBundleSpecAtPath, replace spec.Name with
strings.ToLower(spec.Name) (and likewise normalize any profile identifiers in
spec.Profiles) and then run spec.Validate(manifest) and the duplicate-existence
check against the normalized name stored in the loaded map; do the same
normalization wherever duplicate checks occur (e.g., the other block that
mirrors this logic) so case-only name differences (Foo vs foo) are treated as
duplicates.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: bundle/profile duplicate detection in `LoadBundleSpecs` and `BundleSpec.Validate` is case-sensitive, while activation resolution later matches bundle/profile names with `strings.EqualFold`. That allows case-only duplicates to load successfully and creates ambiguous activation resolution.
- Fix plan: make duplicate detection case-insensitive for bundle names and profile names while preserving the original display values, so `Foo` and `foo` are rejected as duplicates up front.
- Resolution: changed bundle/profile duplicate detection to use case-insensitive lookup keys without rewriting the stored display names.
- Verification: added coverage in `internal/extension/registry_bundles_test.go` and passed `go test ./internal/extension` plus `make verify`.
