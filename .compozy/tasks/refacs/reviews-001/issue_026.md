---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/bridgesdk/webhook.go
line: 262
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYRsm,comment:PRRC_kwDOR5y4QM6-67Ey
---

# Issue 026: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Close error after a successful read triggers a false 413 (Entity Too Large) response.**

When `io.ReadAll` succeeds but `bodyReader.Close()` fails, `errors.Join(nil, closeErr)` returns a non-nil error. The caller at line 206 maps **every** non-nil error from `readBodyWithLimit` to `http.StatusRequestEntityTooLarge`:

```go
// Line 204-208 (unchanged call-site)
body, err := readBodyWithLimit(w, r, config.MaxBodyBytes)
if err != nil {
    http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
    return
}
```

A close failure is not an entity-too-large condition. A valid-sized payload would be rejected with a misleading 413, causing the webhook sender to retry unnecessarily (or permanently drop the delivery if it interprets 413 as permanent).

The fix is to either suppress close errors (with a justification comment per the "never ignore errors" guideline), or surface them through a distinct error type so the call site can decide independently:

<details>
<summary>🐛 Option A — log and suppress (simplest, requires `slog` per guidelines)</summary>

```diff
-func readBodyWithLimit(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, error) {
+func readBodyWithLimit(ctx context.Context, w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, error) {
 	bodyReader := http.MaxBytesReader(w, r.Body, maxBytes)
 	body, err := io.ReadAll(bodyReader)
-	if closeErr := bodyReader.Close(); closeErr != nil {
-		err = errors.Join(err, closeErr)
-	}
+	// Close is best-effort; the HTTP framework also closes r.Body after the handler returns.
+	// A close failure here does not affect the correctness of the body data already read.
+	if closeErr := bodyReader.Close(); closeErr != nil && err == nil {
+		slog.WarnContext(ctx, "bridgesdk: webhook body close error", "err", closeErr)
+	}
 	return body, err
 }
```
</details>

<details>
<summary>🐛 Option B — separate close error so the call site can handle each independently</summary>

```diff
-func readBodyWithLimit(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, error) {
+func readBodyWithLimit(w http.ResponseWriter, r *http.Request, maxBytes int64) (body []byte, readErr error, closeErr error) {
 	bodyReader := http.MaxBytesReader(w, r.Body, maxBytes)
-	body, err := io.ReadAll(bodyReader)
-	if closeErr := bodyReader.Close(); closeErr != nil {
-		err = errors.Join(err, closeErr)
-	}
-	return body, err
+	body, readErr = io.ReadAll(bodyReader)
+	closeErr = bodyReader.Close()
+	return body, readErr, closeErr
 }
```

Then at the call site, handle close errors (log+continue) separately from read errors (413).
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/bridgesdk/webhook.go` around lines 255 - 262, The readBodyWithLimit
function currently joins the io.ReadAll error with bodyReader.Close() which
turns a nil read error plus a close error into a non-nil error that the caller
(the webhook handler that calls readBodyWithLimit) treats as a 413; change
readBodyWithLimit to treat close errors as non-fatal for payload-size handling:
return the read error (if any) but do not return a close-only error—instead log
the closeErr via slog with context (e.g., "closing request body failed" and
include the closeErr and maxBytes) and suppress it from the returned error, so
the caller can reliably map read failures to http.StatusRequestEntityTooLarge
while close failures are recorded but do not trigger a 413.
```

</details>

<!-- fingerprinting:phantom:poseidon:churro -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `readBodyWithLimit` joins `io.ReadAll` and `Close` errors. When the read succeeds and only `Close` fails, the helper returns a non-nil error that the caller currently maps to HTTP `413`, which is the wrong status for a close-only failure.
- Fix plan: treat close-only failures as non-fatal for size enforcement, preserve joined errors when a real read failure also occurs, and adjust tests to lock down the non-fatal close behavior.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
