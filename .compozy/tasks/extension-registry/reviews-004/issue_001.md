---
status: resolved
file: internal/cli/extension_marketplace.go
line: 119
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56486x,comment:PRRC_kwDOR5y4QM63p4Ae
---

# Issue 001: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Reject a nil `Info()` result before using it.**

If a source returns `(*Detail)(nil), nil`, this path falls through and later dereferences `detail.Version`/`detail.Source`, which panics during install.


<details>
<summary>Suggested fix</summary>

```diff
 		detail, err := multi.Info(ctx, slug)
 		if err != nil {
 			return ExtensionRecord{}, err
 		}
+		if detail == nil {
+			return ExtensionRecord{}, fmt.Errorf("cli: extension registry returned no detail for %q", slug)
+		}
```
</details>


Also applies to: 154-156

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/extension_marketplace.go` around lines 113 - 116, The call to
multi.Info may return (nil, nil) and the code then dereferences detail (e.g.,
detail.Version/detail.Source) causing a panic; update the handling after calling
multi.Info in the install flow (where detail variable is used and in the other
similar spot around the second Info call) to explicitly check if detail == nil
and return a descriptive error (e.g., fmt.Errorf("extension info is nil for %s",
slug) or a typed error) before accessing any fields or mapping into
ExtensionRecord, ensuring both occurrences validate the returned pointer.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Root cause analysis: `MultiRegistry.Info()` already converts `(*Detail)(nil), nil` into `registry: package "<slug>" not found`, so this call site never receives a nil detail to dereference.
- Evidence: [`internal/registry/multi.go`](internal/registry/multi.go) lines 132-142 return an error when `detail == nil` and only return a populated detail on success.
- Resolution: No production change was needed. Verified by package tests and the final `make verify` pass.
