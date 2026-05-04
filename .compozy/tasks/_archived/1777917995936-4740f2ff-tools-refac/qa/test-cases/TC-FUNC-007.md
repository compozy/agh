# TC-FUNC-007: Extension Lifecycle Tool Family With Trust-Source And Rollback

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 40 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove `agh__extensions_*` reuses the existing extension manager, registry, marketplace install pipeline, and reconciliation loop. Verify trust-source policy, approval gating, deterministic denial codes, and rollback behavior on failed installs.

## Traceability

- Task: task_08 (Extension Lifecycle Tool Family).
- TechSpec: "Mutable Surface Policy → agh__extensions", "Extensibility Integration Plan", "Implementation Steps".
- ADRs: ADR-004, ADR-006.
- Surfaces: `internal/tools/builtin/extensions.go`, `internal/extension/{manager.go,registry.go,install_managed.go,tool_reconciliation.go}`, `internal/cli/extension.go`, `internal/cli/extension_marketplace.go`, `internal/daemon/extensions.go`.

## Preconditions

- Isolated `AGH_HOME`.
- Fixture marketplace with two known-good extensions and one extension whose install will fail at unpack time.
- Local-source path with one trusted extension and one explicitly-untrusted local path.
- Approval channel auto-approves mutating calls in this lab.

## Test Steps

1. **Search/list/info parity:**
   ```bash
   agh extension search --query qa -o json | tee qa/logs/TC-FUNC-007/cli-search.json
   agh tool invoke agh__extensions_search --input '{"query":"qa"}' -o json | tee qa/logs/TC-FUNC-007/tool-search.json

   agh extension list -o json | tee qa/logs/TC-FUNC-007/cli-list.json
   agh tool invoke agh__extensions_list -o json | tee qa/logs/TC-FUNC-007/tool-list.json

   agh extension info ext-good -o json | tee qa/logs/TC-FUNC-007/cli-info.json
   agh tool invoke agh__extensions_info --input '{"id":"ext-good"}' -o json | tee qa/logs/TC-FUNC-007/tool-info.json
   ```
   - **Expected:** Tool/CLI outputs identical for the same caller scope.

2. **Install + reconciliation:**
   ```bash
   agh tool invoke agh__extensions_install --input '{"id":"ext-good","source":"marketplace","version":"1.0.0"}' -o json \
     | tee qa/logs/TC-FUNC-007/tool-install.json
   ```
   - **Expected:** Extension appears in registry; reconciliation re-projects new tools/skills if any. Install reuses `internal/extension/install_managed.go`.

3. **Update:**
   ```bash
   agh tool invoke agh__extensions_update --input '{"id":"ext-good","version":"1.1.0"}' -o json \
     | tee qa/logs/TC-FUNC-007/tool-update.json
   ```
   - **Expected:** Update applies; rollback path is exercised in Step 5.

4. **Disable / enable:**
   ```bash
   agh tool invoke agh__extensions_disable --input '{"id":"ext-good"}' -o json | tee qa/logs/TC-FUNC-007/tool-disable.json
   agh tool invoke agh__extensions_enable  --input '{"id":"ext-good"}' -o json | tee qa/logs/TC-FUNC-007/tool-enable.json
   ```
   - **Expected:** Reconciliation toggles the runtime activation state. CLI parity matches.

5. **Failed install rollback:**
   ```bash
   agh tool invoke agh__extensions_install --input '{"id":"ext-broken","source":"marketplace","version":"1.0.0"}' -o json \
     | tee qa/logs/TC-FUNC-007/tool-install-broken.json
   ```
   - **Expected:** Returns `EXTENSION_VALIDATION_FAILED` (or equivalent failure during unpack/install). Registry and on-disk state roll back to pre-install state. CLI `agh extension install ext-broken` exhibits the same rollback behavior.

6. **Trust-source denial:**
   ```bash
   agh tool invoke agh__extensions_install --input '{"id":"ext-untrusted","source":"local","path":"/tmp/untrusted"}' -o json \
     | tee qa/logs/TC-FUNC-007/tool-install-untrusted.json
   ```
   - **Expected:** `EXTENSION_SOURCE_FORBIDDEN`.

7. **Approval gating:**
   - Disable approval channel and retry Step 2.
   - **Expected:** `EXTENSION_APPROVAL_REQUIRED`.

8. **Remove:**
   ```bash
   agh tool invoke agh__extensions_remove --input '{"id":"ext-good"}' -o json | tee qa/logs/TC-FUNC-007/tool-remove.json
   ```
   - **Expected:** Registry row gone; reconciliation removes tools/skills the extension provided.

9. **HTTP/UDS parity:** invoke `agh__extensions_install` and `agh__extensions_remove` via UDS for ext-good and confirm matching results.

10. Run focused Go tests:
    ```bash
    go test ./internal/tools/builtin -run "TestExtension" -count=1 | tee qa/logs/TC-FUNC-007/builtin-tests.log
    go test ./internal/extension -count=1 | tee qa/logs/TC-FUNC-007/extension-tests.log
    go test ./internal/daemon -run "TestNativeExtension" -count=1 | tee qa/logs/TC-FUNC-007/daemon-tests.log
    ```

## Evidence To Capture

- All allow / deny / rollback `qa/logs/TC-FUNC-007/*.json`.
- Extension registry sqlite query before/after each step.
- On-disk extension directory listing before/after install + rollback.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Install non-existent ID | `{"id":"missing","source":"marketplace"}` | `EXTENSION_NOT_INSTALLED` (or marketplace-not-found analogue) |
| Update an extension that is not installed | `{"id":"ext-other","version":"1.1.0"}` | `EXTENSION_NOT_INSTALLED` |
| Enable an already-enabled extension | redundant call | Idempotent success or no-op without state regression |
| Rollback after partial extraction | broken-zip fixture | Files removed; registry consistent |

## Channels Exercised

- Tool / CLI / HTTP/UDS.
- Extension registry + on-disk extension directory.

## Related Test Cases

- TC-INT-002 (transport parity).
- TC-FUNC-006 (sister mutable family with same approval/source semantics).
