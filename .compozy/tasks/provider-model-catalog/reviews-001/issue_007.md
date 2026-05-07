---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/api/httpapi/model_catalog_test.go
line: 31
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6sM,comment:PRRC_kwDOR5y4QM6-6brm
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert the `BaseHandlers` wiring too.**

This only verifies the outer `handlers.ModelCatalog` field, so it still passes if `newHandlers` forgets to inject the same service into `BaseHandlers`.


<details>
<summary>Suggested assertion</summary>

```diff
 		if handlers.ModelCatalog != service {
 			t.Fatalf("newHandlers() ModelCatalog = %#v, want %#v", handlers.ModelCatalog, service)
 		}
+		if handlers.BaseHandlers.ModelCatalog != service {
+			t.Fatalf(
+				"newHandlers() BaseHandlers.ModelCatalog = %#v, want %#v",
+				handlers.BaseHandlers.ModelCatalog,
+				service,
+			)
+		}
 	})
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/httpapi/model_catalog_test.go` around lines 21 - 31, The test
only asserts handlers.ModelCatalog but not that the same service was injected
into the embedded BaseHandlers; update the test for newHandlers (using
handlerConfig and httpModelCatalogServiceStub) to also assert BaseHandlers is
non-nil and that handlers.BaseHandlers.ModelCatalog equals the same service
instance (handlers.ModelCatalog or service) to ensure the injection into
BaseHandlers was performed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The dependency test verifies `handlers.ModelCatalog` but not the copy injected into the embedded `BaseHandlers`.
  - `newHandlers(...)` could regress by wiring only the outer field and the current assertion would still pass.
  - Fix: assert `BaseHandlers` remains non-nil and `BaseHandlers.ModelCatalog` points at the same service instance.
