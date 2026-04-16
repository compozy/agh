---
status: resolved
file: internal/api/core/handlers.go
line: 686
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__Bh,comment:PRRC_kwDOR5y4QM63zbx6
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_

## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap bundle-settings errors with operation context before returning.**

The new error path returns `err` directly, which loses call-site context for debugging and logs.

<details>
<summary>💡 Suggested fix</summary>

```diff
 	settings, err := h.Bundles.NetworkSettings(ctx)
 	if err != nil {
-		return nil, err
+		return nil, fmt.Errorf("api: load bundle network settings: %w", err)
 	}
```

</details>

As per coding guidelines, **"Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`."**

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/handlers.go` around lines 683 - 686, The call to
h.Bundles.NetworkSettings returns err directly (in the block around settings,
err := h.Bundles.NetworkSettings(ctx)), losing operation context; change the
return to wrap the error with fmt.Errorf (e.g. return nil, fmt.Errorf("fetching
bundle network settings: %w", err)) and add/import the fmt package if not
already present so the returned error preserves call-site context.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `networkStatusPayload` returns the bundle settings lookup error directly, which drops the handler-level operation context even though the file otherwise wraps transport/service errors with explicit call-site context.
- Fix plan: wrap the `h.Bundles.NetworkSettings(ctx)` failure with an `fmt.Errorf("api: ...: %w", err)` message before returning it.
- Resolution: wrapped the bundle network-settings failure with explicit API operation context.
- Verification: added coverage in `internal/api/core/handlers_internal_test.go` and passed `go test ./internal/api/core` plus `make verify`.
