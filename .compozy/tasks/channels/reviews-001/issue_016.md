---
status: resolved
file: internal/channels/registry.go
line: 186
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLl,comment:PRRC_kwDOR5y4QM623eI-
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`ListInstances` still exposes mutable `DeliveryDefaults` buffers.**

The other read/write paths clone `ChannelInstance`, but this method returns the store slice directly. Callers can mutate `instances[i].DeliveryDefaults` and accidentally leak that mutation back into shared state or later assertions. Please deep-copy each returned instance here as well.

<details>
<summary>Suggested fix</summary>

```diff
 func (s *Service) ListInstances(ctx context.Context) ([]ChannelInstance, error) {
 	if err := s.checkReady(ctx, "list channel instances"); err != nil {
 		return nil, err
 	}
-	return s.store.ListChannelInstances(ctx)
+	instances, err := s.store.ListChannelInstances(ctx)
+	if err != nil {
+		return nil, err
+	}
+	cloned := make([]ChannelInstance, 0, len(instances))
+	for _, instance := range instances {
+		cloned = append(cloned, *cloneChannelInstance(instance))
+	}
+	return cloned, nil
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
// ListInstances returns all persisted channel instances.
func (s *Service) ListInstances(ctx context.Context) ([]ChannelInstance, error) {
	if err := s.checkReady(ctx, "list channel instances"); err != nil {
		return nil, err
	}
	instances, err := s.store.ListChannelInstances(ctx)
	if err != nil {
		return nil, err
	}
	cloned := make([]ChannelInstance, 0, len(instances))
	for _, instance := range instances {
		cloned = append(cloned, *cloneChannelInstance(instance))
	}
	return cloned, nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/channels/registry.go` around lines 181 - 186, ListInstances
currently returns the slice from s.store.ListChannelInstances which exposes
mutable DeliveryDefaults buffers; change ListInstances to deep-copy each
ChannelInstance returned by s.store.ListChannelInstances (iterate over the
returned slice), clone the ChannelInstance struct and allocate/copy the
DeliveryDefaults slice (and any nested mutable slices/maps if present) for each
element, and return the new slice so callers cannot mutate shared state;
reference the ListInstances method, ChannelInstance type, DeliveryDefaults
field, and s.store.ListChannelInstances when making the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `ListInstances()` returns the slice from `ListChannelInstances(...)` directly, so callers can mutate `DeliveryDefaults` buffers on the returned instances.
  - I will deep-clone each returned `ChannelInstance` before returning the slice and add a regression test for mutation isolation.
  - Resolution: `ListInstances()` now returns cloned instances in [internal/channels/registry.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/channels/registry.go:182), with mutation-isolation coverage in [internal/channels/registry_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/channels/registry_test.go:512); verified with `make verify`.
