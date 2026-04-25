---
status: resolved
file: internal/retry/retry_test.go
line: 164
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLii,comment:PRRC_kwDOR5y4QM67SmDp
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Strengthen nil-input error assertions with specific expectations.**

Line 159, Line 162, and Line 184 currently assert only “non-nil error”. Please assert the expected error content so contract regressions are caught.

<details>
<summary>💡 Suggested test assertion tightening</summary>

```diff
 import (
	"context"
	"errors"
+	"strings"
	"testing"
	"time"
 )
@@
-		if err := Do(nilRetryContext(), Policy{}, nil, func(context.Context) error { return nil }); err == nil {
-			t.Fatal("Do(nil context) error = nil, want non-nil")
+		if err := Do(nilRetryContext(), Policy{}, nil, func(context.Context) error { return nil }); err == nil || !strings.Contains(err.Error(), "context is required") {
+			t.Fatalf("Do(nil context) error = %v, want message containing %q", err, "context is required")
		}
-		if err := Do(context.Background(), Policy{}, nil, nil); err == nil {
-			t.Fatal("Do(nil operation) error = nil, want non-nil")
+		if err := Do(context.Background(), Policy{}, nil, nil); err == nil || !strings.Contains(err.Error(), "operation is required") {
+			t.Fatalf("Do(nil operation) error = %v, want message containing %q", err, "operation is required")
		}
@@
-		if err := Wait(nilRetryContext(), 0); err == nil {
-			t.Fatal("Wait(nil context) error = nil, want non-nil")
+		if err := Wait(nilRetryContext(), 0); err == nil || !strings.Contains(err.Error(), "context is required") {
+			t.Fatalf("Wait(nil context) error = %v, want message containing %q", err, "context is required")
		}
```
</details>

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".


Also applies to: 184-186

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/retry/retry_test.go` around lines 159 - 164, The tests currently
only assert that Do(...) returns a non-nil error; update them to assert specific
error content using ErrorContains/ErrorAs (or the testing helper used in the
repo) so regressions are caught: for the call Do(nilRetryContext(), Policy{},
nil, func(...){...}) assert the error contains the expected "nil context"
message or sentinel error, and for Do(context.Background(), Policy{}, nil, nil)
assert the error contains the expected "nil operation" message (and make the
same stronger assertion for the similar checks at lines 184-186). Locate the
assertions around the Do function calls, nilRetryContext, and Policy in
retry_test.go and replace the generic non-nil checks with precise
ErrorContains/ErrorAs assertions referencing the expected error text or sentinel
error values.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: nil-input retry tests only check for non-nil errors, so message/contract regressions for nil contexts or nil operations would still pass.
- Fix approach: assert specific error content for nil context and nil operation paths in `Do` and `Wait`.
