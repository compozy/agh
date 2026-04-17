---
status: resolved
file: internal/bundles/service.go
line: 300
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57znot,comment:PRRC_kwDOR5y4QM645ilW
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Skip bundle-resource loading when there are no activations.**

This preload changes the empty-state behavior: `ListActivations` can now fail on an otherwise empty system if `ListBundleResources` is down. Returning early on `len(activations) == 0` avoids that regression and keeps the optimization for non-empty lists.


<details>
<summary>💡 Proposed fix</summary>

```diff
 	activations, err := s.store.ListBundleActivations(ctx)
 	if err != nil {
 		return nil, fmt.Errorf("bundles: list bundle activations: %w", err)
 	}
+	if len(activations) == 0 {
+		return []ActivationPreview{}, nil
+	}
 	bundleRecords, err := s.store.ListBundleResources(ctx)
 	if err != nil {
 		return nil, fmt.Errorf("bundles: list bundle resources for activations: %w", err)
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/bundles/service.go` around lines 296 - 300, In ListActivations,
avoid preloading bundle resources when there are no activations by adding an
early return if len(activations) == 0 before calling
s.store.ListBundleResources; specifically, move or guard the call to
s.store.ListBundleResources and the creation of bundleLookup
(newBundleRecordLookup(bundleRecords)) so they only run when activations is
non-empty, returning the empty result immediately to prevent ListBundleResources
failures from affecting empty-state ListActivations.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `ListActivations` always preloads bundle resources immediately after listing activations, so an empty activation set can still fail when `ListBundleResources` is unavailable.
- Fix plan: return early when `len(activations) == 0` and add a regression test that verifies an empty-state listing does not depend on bundle-resource loading.
- Resolution: `ListActivations` now returns an empty slice before loading bundle resources, and `TestServiceListActivationsReturnsEmptyWithoutLoadingBundleResources` covers the empty-state regression.
- Verification: `go test ./internal/bundles ./internal/environment/daytona ./internal/extension ./internal/tools` and `make verify` passed on 2026-04-17.
