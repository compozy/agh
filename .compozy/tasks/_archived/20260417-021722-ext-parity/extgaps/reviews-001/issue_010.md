---
status: resolved
file: internal/daemon/bridges.go
line: 194
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__Bv,comment:PRRC_kwDOR5y4QM63zbyP
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# Find BridgeSecretBinding definition
rg "type BridgeSecretBinding" --type go -A 10
```

Repository: compozy/agh

Length of output: 1341

---

🏁 Script executed:

```shell
# Check the store method implementations
fd "\.go$" | xargs rg "PutBridgeSecretBinding|ListBridgeSecretBindings|DeleteBridgeSecretBinding" --type go -B 2 -A 15
```

Repository: compozy/agh

Length of output: 50367

---

🏁 Script executed:

```shell
# Look at the file structure around the reviewed code
head -200 internal/daemon/bridges.go | tail -100
```

Repository: compozy/agh

Length of output: 3416

---

🏁 Script executed:

```shell
# Check if store methods wrap errors
rg "PutBridgeSecretBinding|ListBridgeSecretBindings|DeleteBridgeSecretBinding" internal/store/globaldb/global_db_bridge.go -A 30 | head -150
```

Repository: compozy/agh

Length of output: 3420

---

🏁 Script executed:

```shell
# Check how the contract layer converts and validates bindings
rg "ToBridgeSecretBinding" internal/api/contract/bridges.go -A 15
```

Repository: compozy/agh

Length of output: 862

---

</details>

**Normalize binding fields on write and wrap store errors with daemon operation context.**

`PutSecretBinding` forwards the binding struct fields unchanged; `ListSecretBindings` and `DeleteSecretBinding` trim their string parameters before store operations. This creates a key-matching hazard: if `BridgeInstanceID` or `BindingName` contain whitespace when persisted, subsequent list/delete calls that trim will fail to match those rows. Additionally, all three methods return store errors directly, losing the daemon-layer operation context. Trim the binding's `BridgeInstanceID` and `BindingName` fields before persistence, and wrap all store errors with `fmt.Errorf("context: %w", err)` to retain both stack context and `errors.Is`/`errors.As` semantics at the daemon boundary.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/bridges.go` around lines 148 - 194, ListSecretBindings,
PutSecretBinding, and DeleteSecretBinding in bridgeRuntime must normalize
binding keys and wrap store errors: trim BridgeInstanceID and BindingName on
write by updating PutSecretBinding to sanitize binding.BridgeInstanceID and
binding.BindingName (bridgepkg.BridgeSecretBinding) before calling
r.store.PutBridgeSecretBinding, and ensure any call to
r.store.ListBridgeSecretBindings, r.store.PutBridgeSecretBinding, and
r.store.DeleteBridgeSecretBinding wraps returned errors with fmt.Errorf("daemon:
<operation> failed: %w", err) (use appropriate operation names) while keeping
the existing trimming of parameters on ListSecretBindings/DeleteSecretBinding to
avoid key-mismatch.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `PutSecretBinding` persists `BridgeInstanceID` and `BindingName` without trimming, while list/delete trim their inputs before querying. That can create whitespace-key mismatches across write/read/delete paths. The three methods also return raw store errors, which loses daemon-layer operation context.
- Fix plan: normalize the binding keys on write and wrap all secret-binding store failures with daemon operation context.
- Resolution: normalized secret-binding keys before persistence and wrapped list/put/delete failures with daemon context.
- Verification: added coverage in `internal/daemon/bridges_test.go` and passed `go test ./internal/daemon` plus `make verify`.
