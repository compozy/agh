---
status: resolved
file: internal/daemon/hooks_bridge.go
line: 436
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAaV,comment:PRRC_kwDOR5y4QM62zlsg
---

# Issue 017: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap declaration-provider failures with source context.**

These returns bubble provider errors up raw, so a later `hooks.Rebuild()` failure loses whether it came from the chained provider or the extension runtime. Adding `fmt.Errorf(...: %w)` here will make extension hook boot failures much easier to diagnose.


<details>
<summary>Suggested fix</summary>

```diff
 func chainDeclarationProviders(providers ...hookspkg.DeclarationProvider) hookspkg.DeclarationProvider {
 	return func(ctx context.Context) ([]hookspkg.HookDecl, error) {
 		chained := make([]hookspkg.HookDecl, 0, len(providers))
-		for _, provider := range providers {
+		for idx, provider := range providers {
 			if provider == nil {
 				continue
 			}
 
 			decls, err := provider(ctx)
 			if err != nil {
-				return nil, err
+				return nil, fmt.Errorf("daemon: load hook declarations from provider %d: %w", idx, err)
 			}
 			chained = append(chained, decls...)
 		}
 		return chained, nil
 	}
 }
 
 func extensionDeclarationProvider(getRuntime func() extensionRuntime) hookspkg.DeclarationProvider {
 	return func(ctx context.Context) ([]hookspkg.HookDecl, error) {
 		if getRuntime == nil {
 			return nil, nil
@@
 		runtime := getRuntime()
 		if runtime == nil {
 			return nil, nil
 		}
-		return runtime.HookDeclarations(ctx)
+		decls, err := runtime.HookDeclarations(ctx)
+		if err != nil {
+			return nil, fmt.Errorf("daemon: load hook declarations from extension runtime: %w", err)
+		}
+		return decls, nil
 	}
 }
```
</details>
As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".


Also applies to: 450-454

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/hooks_bridge.go` around lines 434 - 436, The provider call
returns raw errors that lose source context; for both places where you call
decls, err := provider(ctx) (and the similar block around lines 450-454), wrap
returned errors with fmt.Errorf to add context before returning (e.g., return
nil, fmt.Errorf("provider %s failed: %w", providerNameOrSource, err)) so that
failures from the chained declaration provider are distinguishable from
extension/runtime errors and later hooks.Rebuild() failures include the original
source.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: Hook declaration providers currently bubble raw errors, which makes later `hooks.Rebuild()` failures lose the source that failed. I will wrap provider errors with contextual source information and add tests for the wrapped paths.
