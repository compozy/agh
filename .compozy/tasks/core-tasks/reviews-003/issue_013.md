---
status: resolved
file: internal/network/audit.go
line: 147
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM565Hzm,comment:PRRC_kwDOR5y4QM63qGbA
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, locate and examine the audit.go file
find . -name "audit.go" -type f | head -20
```

Repository: compozy/agh

Length of output: 82

---

🏁 Script executed:

```shell
# Read the relevant lines from internal/network/audit.go
if [ -f "internal/network/audit.go" ]; then
  echo "=== Lines 130-155 ==="
  sed -n '130,155p' internal/network/audit.go | cat -n
  echo ""
  echo "=== Lines 280-295 ==="
  sed -n '280,295p' internal/network/audit.go | cat -n
else
  echo "File not found"
fi
```

Repository: compozy/agh

Length of output: 1530

---

🏁 Script executed:

```shell
# Get the full file size to understand context better
wc -l internal/network/audit.go
```

Repository: compozy/agh

Length of output: 84

---

🏁 Script executed:

```shell
# Let me get more context around these lines to understand the function names
sed -n '120,160p' internal/network/audit.go | cat -n
```

Repository: compozy/agh

Length of output: 1663

---

🏁 Script executed:

```shell
# Also check around lines 287-289 for more context
sed -n '275,300p' internal/network/audit.go | cat -n
```

Repository: compozy/agh

Length of output: 1132

---



</details>

**Wrap propagated errors with operation context.**

Two locations in this function return or join raw errors without local operation context:

1. **Line 139** (`return err`): The error from `normalizeTaskIngressAuditEntry()` should be wrapped with operation context before returning.
2. **Lines 144 and 147**: Errors from `w.appendFile(entry)` and `w.store.WriteNetworkAudit(ctx, entry)` should be wrapped before joining.
3. **Line 289**: The error from `entry.Validate()` should be wrapped before returning.

Wrap each error with `fmt.Errorf()` to add operation-specific context (e.g., `fmt.Errorf("network: normalize task ingress audit entry: %w", err)`), making root-cause tracing easier in production.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/audit.go` around lines 136 - 147, The errors returned from
normalizeTaskIngressAuditEntry, w.appendFile, w.store.WriteNetworkAudit, and
entry.Validate need to be wrapped with operation context using fmt.Errorf and %w
before being returned or joined; specifically, replace the raw return of err
from normalizeTaskIngressAuditEntry with a wrapped error like
fmt.Errorf("network: normalize task ingress audit entry: %w", err), wrap the
results of w.appendFile(entry) and w.store.WriteNetworkAudit(ctx, entry)
similarly before they are combined into recordErr (so recordErr =
errors.Join(recordErr, fmt.Errorf("network: append file audit entry: %w", err)))
and wrap the error returned from entry.Validate() with context (e.g.,
fmt.Errorf("network: validate audit entry: %w", err)) so all propagated errors
include clear operation-specific context.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `RecordTaskIngress` and `normalizeTaskIngressAuditEntry` currently return/join raw downstream errors without local operation context, which weakens root-cause tracing. I will wrap the normalization, file append, store write, and entry validation errors with operation-specific context and add regression coverage.
  Resolution: Wrapped task-ingress normalization, append, persistence, and entry-validation failures with local operation context and added regression tests for normalization and sink-write failure paths.
