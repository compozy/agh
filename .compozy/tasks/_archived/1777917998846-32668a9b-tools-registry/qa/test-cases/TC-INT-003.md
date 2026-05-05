# TC-INT-003 — Toolset lifecycle: install → enable → invoke → disable → remove

- **Priority:** P2
- **Type:** Integration / extension lifecycle
- **Trace:** Task 06, Task 07

## Test Steps

1. Install extension publishing toolset `linear__read` and tools.
   - **Expected:** Tools and toolset appear in operator views with cold descriptors.
2. Enable extension.
   - **Expected:** Runtime reconciliation marks tools executable; session projection updated.
3. Invoke a tool from the toolset.
   - **Expected:** Successful call.
4. Disable extension.
   - **Expected:** Tools become unavailable with `extension_inactive`; session projection drops them.
5. Remove extension.
   - **Expected:** Cold descriptors removed.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/extension -run TestExtensionToolLifecycle`
