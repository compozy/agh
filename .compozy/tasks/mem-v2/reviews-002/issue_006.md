---
provider: coderabbit
pr: "108"
round: 2
round_created_at: 2026-05-06T04:43:32.489895Z
status: resolved
file: internal/cli/memory.go
line: 1035
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2cOw,comment:PRRC_kwDOR5y4QM6-Uf95
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Make `--timeout` affect the drain call or drop the flag.**

This command only validates the duration string; it never derives a timed context from it. Users can pass `--timeout=5s` and still wait indefinitely if the parent context has no deadline.




<details>
<summary>Suggested fix</summary>

```diff
+import "context"
@@
 	RunE: func(cmd *cobra.Command, _ []string) error {
 		client, err := clientFromDeps(deps)
 		if err != nil {
 			return err
 		}
+		ctx := cmd.Context()
 		if strings.TrimSpace(timeoutRaw) != "" {
-			if _, err := time.ParseDuration(strings.TrimSpace(timeoutRaw)); err != nil {
+			timeout, err := time.ParseDuration(strings.TrimSpace(timeoutRaw))
+			if err != nil {
 				return fmt.Errorf("memory.extractor.timeout_invalid: %w", err)
 			}
+			var cancel context.CancelFunc
+			ctx, cancel = context.WithTimeout(ctx, timeout)
+			defer cancel()
 		}
-		response, err := client.DrainMemoryExtractor(cmd.Context())
+		response, err := client.DrainMemoryExtractor(ctx)
 		if err != nil {
 			return err
 		}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/cli/memory.go` around lines 1010 - 1035, The --timeout flag is only
validated but not applied; update newMemoryExtractorDrainCommand to parse
timeoutRaw into a time.Duration, derive a timed context via
context.WithTimeout(ctx, duration) (remember to defer cancel()), and pass that
timed context to client.DrainMemoryExtractor instead of cmd.Context(); keep
validating the parse error as already done and ensure the default "60s" is
honored when timeoutRaw is empty or whitespace. Reference symbols:
newMemoryExtractorDrainCommand, timeoutRaw, client.DrainMemoryExtractor,
cmd.Context(), and the deferred cancel for the created context.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `newMemoryExtractorDrainCommand` validates `--timeout` but never derives a timed context, so the call still uses the parent context and can wait indefinitely.
- Evidence: the command parses the duration and then always passes `cmd.Context()` into `client.DrainMemoryExtractor`.
- Fix plan: parse the timeout into a `context.WithTimeout` child context and pass that context to the drain call.
- Resolution: implemented in `internal/cli/memory.go` and validated with targeted CLI regression coverage in `internal/cli/memory_test.go`, which was the minimal extra file needed to prove the timeout context behavior.
- Verification: targeted `go test` for `internal/cli` passed, and fresh `make verify` passed on 2026-05-06.
