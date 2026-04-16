---
status: resolved
file: internal/daemon/automation_resources.go
line: 121
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQw,comment:PRRC_kwDOR5y4QM64dqG-
---

# Issue 040: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Fail fast on partial resource wiring**

Returning success when only one of `raw`/`codecs` is nil can silently disable automation resource projection due to misconfiguration.



<details>
<summary>Suggested guard</summary>

```diff
 func automationResourceStores(
 	raw resources.RawStore,
 	codecs *resources.CodecRegistry,
 ) (
@@
 ) {
-	if raw == nil || codecs == nil {
+	if raw == nil && codecs == nil {
 		return nil, nil, nil
 	}
+	if raw == nil || codecs == nil {
+		return nil, nil, errors.New("daemon: raw store and codec registry must be provided together")
+	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if raw == nil && codecs == nil {
		return nil, nil, nil
	}
	if raw == nil || codecs == nil {
		return nil, nil, errors.New("daemon: raw store and codec registry must be provided together")
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/automation_resources.go` around lines 119 - 121, The current
guard returns (nil, nil, nil) when either raw or codecs is nil, which can
silently disable automation resource projection; change the check to fail fast:
if raw == nil || codecs == nil return an explicit error (not nil error)
describing which of raw/codecs is missing (use the raw and codecs symbols to
detect which is nil) so callers can surface configuration issues; update the
surrounding function's return path (e.g., the function that currently returns
(raw, codecs, nil)) to propagate this non-nil error and adjust callers if
necessary.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `automationResourceStores` still returns `(nil, nil, nil)` when either `raw` or `codecs` is missing, which silently disables automation resource projection on partial wiring mistakes. The function should only return nil stores when both dependencies are absent; otherwise it needs to fail fast with a clear configuration error. Coverage for the partial-miswire cases requires a minimal out-of-scope unit test in `internal/daemon/automation_resources_test.go` because the scoped daemon test file is integration-tagged and would not run under the normal verification gate.
