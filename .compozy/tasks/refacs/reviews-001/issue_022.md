---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/bridgesdk/errors.go
line: 319
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYRsf,comment:PRRC_kwDOR5y4QM6-67Es
---

# Issue 022: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

<details>
<summary>❓ Verification inconclusive</summary>

Script executed:

```shell
find . -name "errors.go" -path "*/bridgesdk/*"
```

Repository: compozy/agh

---

Repository: compozy/agh
Exit code: 0


stdout:
```
./internal/bridgesdk/errors.go

```

---

Script executed:

```shell
cat -n internal/bridgesdk/errors.go | sed -n '310,360p'
```

Repository: compozy/agh

---

Repository: compozy/agh
Exit code: 0


stdout:
```
   310	
   311	// RetryDo retries the operation according to the shared classification policy.
   312	func RetryDo[T any](ctx context.Context, config RetryConfig, fn func(context.Context) (T, error)) (T, error) {
   313		var zero T
   314		if ctx == nil {
   315			return zero, errors.New("bridgesdk: retry context is required")
   316		}
   317		if fn == nil {
   318			return zero, errors.New("bridgesdk: retry function is required")
   319		}
   320		if config.Attempts <= 0 {
   321			config.Attempts = 1
   322		}
   323		if config.MinDelay <= 0 {
   324			config.MinDelay = 300 * time.Millisecond
   325		}
   326		if config.MaxDelay <= 0 {
   327			config.MaxDelay = 30 * time.Second
   328		}
   329		if config.RandFloat == nil {
   330			config.RandFloat = rand.Float64
   331		}
   332	
   333		for attempt := 1; attempt <= config.Attempts; attempt++ {
   334			result, err := fn(ctx)
   335			if err == nil {
   336				return result, nil
   337			}
   338	
   339			classified := ClassifyError(err)
   340			recovery := classified.Recovery()
   341			if !recovery.Retry || attempt == config.Attempts {
   342				return zero, err
   343			}
   344	
   345			delay := retryDelay(config, attempt, recovery)
   346			if config.OnRetry != nil {
   347				config.OnRetry(attempt, config.Attempts, classified)
   348			}
   349	
   350			if err := retrypkg.Wait(ctx, delay); err != nil {
   351				return zero, fmt.Errorf("bridgesdk: wait before retry: %w", err)
   352			}
   353		}
   354	
   355		return zero, errors.New("bridgesdk: retry attempts exhausted")
   356	}
   357	
   358	func retryDelay(config RetryConfig, attempt int, recovery RecoveryDecision) time.Duration {
   359		if recovery.RetryAfter > 0 {
   360			return recovery.RetryAfter

```

---



</details>

**Add `ctx.Err()` check before entering the retry loop to respect context cancellation.**

A nil context passes the guard at line 314, but a pre-canceled or expired context is not detected until line 350 (during `Wait`). On the first attempt, `fn(ctx)` executes on line 334 even if the caller has already aborted, violating context discipline. Insert `if err := ctx.Err(); err != nil { return zero, err }` after the config defaults and before the loop.

Additionally, line 355's exhaustion error is unreachable—the loop always returns on the final attempt at line 342. Remove or replace it with a panic to signal a code invariant violation.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/bridgesdk/errors.go` around lines 313 - 319, After applying defaults
but before entering the retry loop in the retry function (the block that
declares var zero T and uses ctx and fn and calls Wait), check the context for
cancellation by doing if err := ctx.Err(); err != nil { return zero, err } so we
don't invoke fn(ctx) when the caller already canceled; also remove the
unreachable “exhaustion” error that can never be returned on the final loop
iteration and replace it with a panic (or runtime assertion) to signal a code
invariant violation instead of dead code — target the retry loop, ctx, fn, Wait
and the zero T return sites when making these changes.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `RetryDo` validates `ctx != nil` but still enters the retry loop and invokes `fn(ctx)` when `ctx.Err()` is already non-nil. The fallback `retry attempts exhausted` return is also unreachable because the loop always returns on the final attempt.
- Fix plan: check `ctx.Err()` after config normalization and before the retry loop, then replace the unreachable tail return with an explicit invariant panic.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
