---
status: resolved
file: internal/daemon/channels.go
line: 173
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tkbg,comment:PRRC_kwDOR5y4QM624L_N
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Resolve bound secrets before persisting `starting`.**

If `resolveBoundSecrets` fails here, the instance is left stored as `starting` even though launch never actually happened. That moves lifecycle state ahead of reality and can strand the instance in the wrong status.



<details>
<summary>Suggested direction</summary>

```diff
-	launching, err := r.transitionInstance(ctx, instance.ID, true, channelspkg.ChannelStatusStarting, false, "launch")
-	if err != nil {
-		return nil, err
-	}
-
-	boundSecrets, err := r.resolveBoundSecrets(ctx, launching.ID)
+	boundSecrets, err := r.resolveBoundSecrets(ctx, instance.ID)
 	if err != nil {
-		return nil, fmt.Errorf("daemon: resolve bound secrets for channel instance %q: %w", launching.ID, err)
+		return nil, fmt.Errorf("daemon: resolve bound secrets for channel instance %q: %w", instance.ID, err)
+	}
+
+	launching, err := r.transitionInstance(ctx, instance.ID, true, channelspkg.ChannelStatusStarting, false, "launch")
+	if err != nil {
+		return nil, err
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	boundSecrets, err := r.resolveBoundSecrets(ctx, instance.ID)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve bound secrets for channel instance %q: %w", instance.ID, err)
	}

	launching, err := r.transitionInstance(ctx, instance.ID, true, channelspkg.ChannelStatusStarting, false, "launch")
	if err != nil {
		return nil, err
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/channels.go` around lines 165 - 173, The code currently calls
transitionInstance(..., channelspkg.ChannelStatusStarting, ...) before
resolveBoundSecrets, which can persist the instance as Starting even if
resolveBoundSecrets fails; change the call order so resolveBoundSecrets(ctx,
instance.ID) is executed and succeeds before invoking transitionInstance to
persist the Starting state (or modify transitionInstance usage to delay
persistence until after bound secrets are resolved), ensuring that
transitionInstance/launching is only created after resolveBoundSecrets returns
successfully and preserving consistency between launching/launching.ID and
resolved secrets.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `Valid`
- Notes:
  `ResolveChannelRuntime` currently transitions the instance to `starting` before resolving bound secrets. If bound-secret resolution fails, the durable lifecycle state moves ahead of reality and the instance can be stranded as `starting` even though launch never produced a runtime payload.
  Resolved in `internal/daemon/channels.go` by resolving bound secrets against the existing instance first and persisting the `starting` transition only after secret resolution succeeds. Regression coverage was added in `internal/daemon/channels_test.go`. Verified with `go test ./internal/daemon -count=1`, `go test -tags integration ./internal/daemon -count=1`, and the final `make verify` pass.
