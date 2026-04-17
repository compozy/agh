---
status: resolved
file: internal/daemon/tool_mcp_resources_test.go
line: 476
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEb_,comment:PRRC_kwDOR5y4QM640q0Z
---

# Issue 015: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Avoid `strings.HasPrefix` on error messages for assertions.**

Per coding guidelines, use `errors.Is()` or `errors.As()` for error matching. If the production code wraps errors with `fmt.Errorf("daemon: resolve tool codec: %w", err)`, tests should match the underlying error type rather than parsing message prefixes.



<details>
<summary>🔧 Suggested approach</summary>

If the daemon package defines sentinel errors or typed errors for codec resolution failures:

```go
// In production code:
var ErrToolCodecResolution = errors.New("tool codec resolution failed")

// Wrap when returning:
return nil, fmt.Errorf("daemon: resolve tool codec: %w", ErrToolCodecResolution)

// In test:
if !errors.Is(err, ErrToolCodecResolution) {
    t.Fatalf("expected tool codec resolution error, got: %v", err)
}
```

Alternatively, if checking for a specific error type:
```go
var codecErr *CodecResolutionError
if !errors.As(err, &codecErr) {
    t.Fatalf("expected CodecResolutionError, got: %T", err)
}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/tool_mcp_resources_test.go` around lines 450 - 476, The tests
in newToolMCPPublisher currently assert error messages with strings.HasPrefix;
instead update the two assertions to use errors.Is or errors.As against the
package sentinel/typed errors (e.g., ErrToolCodecResolution and
ErrMCPCodecResolution or a *CodecResolutionError) returned by
newToolMCPPublisher via bootState.resourceCodecs; locate the failing checks
around calls to daemon.newToolMCPPublisher (the "empty codecs" and "tool-only
codecs" cases) and replace the string prefix checks with errors.Is(err,
ErrToolCodecResolution) and errors.Is(err, ErrMCPCodecResolution) (or use
errors.As to assert the concrete *CodecResolutionError), and ensure production
returns wrap the underlying sentinel/typed errors with fmt.Errorf("%s: %w", ...
) so the test can match them with errors.Is/errors.As.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `newToolMCPPublisher` wraps codec resolution failures with `%w`, but the tests still inspect message prefixes instead of matching the underlying `resources.ErrCodecNotFound`.
- Fix plan: replace the prefix assertions with `errors.Is(err, resources.ErrCodecNotFound)`.
- Resolution: replaced the string-prefix assertions with `errors.Is(err, resources.ErrCodecNotFound)`.
- Verification: `go test ./internal/daemon` passed. Historical note: the earlier `driver/dist/index.js` blocker was stale; the shipped mock driver is `internal/testutil/acpmock/cmd/acpmock-driver`.
