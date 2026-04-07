---
status: resolved
file: internal/httpapi/helpers_test.go
line: 161
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCN,comment:PRRC_kwDOR5y4QM61T6Hm
---

# Issue 014: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Consider a more appropriate default for `Register`.**

The `Register` method creates a new workspace, so returning `ErrWorkspaceNotFound` as the default (when no hook is set) is semantically inconsistent. Other stubs like `Unregister`, `Update`, and `Get` appropriately return this error since they operate on existing workspaces. For `Register`, consider returning a zero value with `nil` error, or a distinct error like `errors.New("stubWorkspaceService: Register not implemented")`.


<details>
<summary>Proposed fix</summary>

```diff
 func (s stubWorkspaceService) Register(ctx context.Context, opts workspacepkg.RegisterOptions) (workspacepkg.Workspace, error) {
 	if s.registerFn != nil {
 		return s.registerFn(ctx, opts)
 	}
-	return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
+	return workspacepkg.Workspace{}, nil
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (s stubWorkspaceService) Register(ctx context.Context, opts workspacepkg.RegisterOptions) (workspacepkg.Workspace, error) {
	if s.registerFn != nil {
		return s.registerFn(ctx, opts)
	}
	return workspacepkg.Workspace{}, nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/httpapi/helpers_test.go` around lines 156 - 161, The
stubWorkspaceService.Register default behavior is semantically wrong: instead of
returning workspacepkg.ErrWorkspaceNotFound when no registerFn is provided,
change Register (method name: Register on type stubWorkspaceService) to return a
zero-value workspace and nil error or a distinct sentinel error (e.g.,
errors.New("stubWorkspaceService: Register not implemented")); update the
function body so when s.registerFn == nil it returns the chosen default (zero
workspace with nil or the new sentinel) rather than
workspacepkg.ErrWorkspaceNotFound to reflect that Register creates resources
rather than looking them up.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  - The default `ErrWorkspaceNotFound` return is intentional test scaffolding.
  - When a test forgets to provide `registerFn`, returning a not-found error makes the unexpected call fail loudly instead of silently succeeding with a zero-value workspace.
  - Changing the default to `nil` would weaken the stub and make accidental gaps in test setup harder to detect.
