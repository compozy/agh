---
status: resolved
file: internal/channels/registry.go
line: 166
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLk,comment:PRRC_kwDOR5y4QM623eI9
---

# Issue 015: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap store failures with operation context.**

This new service is returning raw persistence errors here, which makes it hard to tell whether a failure came from create, update, route resolution, or validation once it bubbles up through the daemon/API layers. Please wrap store/helper errors with action-specific context in this method and the sibling registry methods.

<details>
<summary>Suggested fix</summary>

```diff
 	instance, err := req.toInstance(s.now)
 	if err != nil {
-		return nil, err
+		return nil, fmt.Errorf("channels: build channel instance: %w", err)
 	}
 	if err := s.store.InsertChannelInstance(ctx, instance); err != nil {
-		return nil, err
+		return nil, fmt.Errorf("channels: insert channel instance %q: %w", instance.ID, err)
 	}
```
</details>
As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (s *Service) CreateInstance(ctx context.Context, req CreateInstanceRequest) (*ChannelInstance, error) {
	if err := s.checkReady(ctx, "create channel instance"); err != nil {
		return nil, err
	}

	instance, err := req.toInstance(s.now)
	if err != nil {
		return nil, fmt.Errorf("channels: build channel instance: %w", err)
	}
	if err := s.store.InsertChannelInstance(ctx, instance); err != nil {
		return nil, fmt.Errorf("channels: insert channel instance %q: %w", instance.ID, err)
	}
	return cloneChannelInstance(instance), nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/channels/registry.go` around lines 153 - 166, In CreateInstance,
wrap underlying errors with action-specific context instead of returning raw
errors: for failures from s.checkReady, req.toInstance, and
s.store.InsertChannelInstance, return fmt.Errorf("create channel instance: %w",
err) (or include more granular context like "create channel instance:
validation: %w" for req.toInstance and "create channel instance: store insert:
%w" for s.store.InsertChannelInstance) so callers can distinguish create
failures; apply the same wrapping pattern to sibling registry methods that call
checkReady, toInstance, and store operations (retain use of cloneChannelInstance
for the success return).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `CreateInstance` and nearby registry methods currently leak raw validation/store errors, which loses the operation context once the error reaches API/daemon layers.
  - I will wrap the relevant registry errors with action-specific context while preserving `errors.Is` behavior.
  - Resolution: Wrapped registry validation/store errors with operation-specific context in [internal/channels/registry.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/channels/registry.go:153) and added a create-error regression in [internal/channels/registry_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/channels/registry_test.go:488); verified with `make verify`.
