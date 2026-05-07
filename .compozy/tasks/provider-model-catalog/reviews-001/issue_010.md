---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/cli/install_test.go
line: 179
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6sS,comment:PRRC_kwDOR5y4QM6-6br1
---

# Issue 010: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert provider existence before checking nested default model.**

This assertion can pass even if `"blackbox"` is absent, since map lookup returns a zero-value config. Verify presence first, then assert `Models.Default`.
 
<details>
<summary>Suggested test hardening</summary>

```diff
-		if got := cfg.Providers["blackbox"].Models.Default; got != "" {
+		blackboxCfg, ok := cfg.Providers["blackbox"]
+		if !ok {
+			t.Fatal(`cfg.Providers["blackbox"] missing`)
+		}
+		if got := blackboxCfg.Models.Default; got != "" {
 			t.Fatalf("cfg.Providers[blackbox].Models.Default = %q, want empty", got)
 		}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		blackboxCfg, ok := cfg.Providers["blackbox"]
		if !ok {
			t.Fatal(`cfg.Providers["blackbox"] missing`)
		}
		if got := blackboxCfg.Models.Default; got != "" {
			t.Fatalf("cfg.Providers[blackbox].Models.Default = %q, want empty", got)
		}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/cli/install_test.go` around lines 178 - 179, The test currently
asserts cfg.Providers["blackbox"].Models.Default without first ensuring the
"blackbox" provider exists, which can hide missing-provider bugs; update the
test in install_test.go to first check that cfg.Providers contains the
"blackbox" key (e.g., assert presence or use a lookup like val, ok :=
cfg.Providers["blackbox"] and fail if !ok), then use that retrieved provider
value to assert val.Models.Default is empty; reference cfg, Providers, the
"blackbox" key, and Models.Default when making the changes.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - Strengthening the assertion exposed that provider-managed bootstrap intentionally persists `defaults.provider = "blackbox"` without forcing an empty `[providers.blackbox.models]` table into the config file.
  - Reloaded global config therefore omits the explicit map entry while still preserving the intended runtime behavior: the selected provider is `blackbox`, and the provider-managed model remains empty by default.
  - Because the stronger assertion conflicts with the current bootstrap contract rather than exposing a defect in it, this review item is not valid for this batch.
