---
status: resolved
file: internal/daemon/section_selector.go
line: 40
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dM6,comment:PRRC_kwDOR5y4QM65IPEO
---

# Issue 020: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep the selection pass even when startup resolution is unavailable.**

This early return skips the `Provider == nil` filter and the name de-duplication below, so `Select` can return invalid or duplicate sections whenever the selector is nil or resolverless.

<details>
<summary>Suggested fix</summary>

```diff
     normalized := normalizeAndSortPromptSectionDescriptors(descriptors)
-    if s == nil || s.resolver == nil {
-        return normalized, ResolvedHarnessContext{}, nil
-    }
+    resolved := ResolvedHarnessContext{}
+    if s == nil || s.resolver == nil {
+        selected := make([]PromptSectionDescriptor, 0, len(normalized))
+        seen := make(map[string]struct{}, len(normalized))
+        for _, descriptor := range normalized {
+            if descriptor.Provider == nil {
+                continue
+            }
+            if _, exists := seen[descriptor.Name]; exists {
+                continue
+            }
+            seen[descriptor.Name] = struct{}{}
+            selected = append(selected, descriptor)
+        }
+        return selected, resolved, nil
+    }

-    resolved, err := s.resolver.ResolveStartup(startup)
+    var err error
+    resolved, err = s.resolver.ResolveStartup(startup)
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/section_selector.go` around lines 38 - 40, The early return
in Select when s == nil || s.resolver == nil skips the Provider == nil filter
and name de-duplication; remove that early return and instead short-circuit only
the startup resolution logic: keep executing the rest of Select (so the
Provider==nil filter and the de-duplication logic still run) and, when s or
s.resolver is nil, treat resolved context as an empty ResolvedHarnessContext or
skip calls to s.resolver.ResolveStartup but do not return; update the code paths
in Select that reference s.resolver (e.g., any call to
s.resolver.ResolveStartup) to be guarded by a nil check so resolution is skipped
safely while still applying Provider filtering and name de-duplication.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `Select` returns early when the selector or resolver is nil, which skips the later `Provider == nil` filter and the name de-duplication pass.
  - That means callers can receive unusable or duplicate section descriptors precisely in the fallback path that should remain safe.
  - I will keep the filtering/de-duplication logic active even when startup resolution is unavailable and add a regression test for the fallback behavior.
