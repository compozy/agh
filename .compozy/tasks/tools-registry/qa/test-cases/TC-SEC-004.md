# TC-SEC-004 — Mutating tool mislabeled `read_only` is rejected at descriptor validation

- **Priority:** P0
- **Type:** Security / risk classification
- **Trace:** Task 01 (descriptor validation), Task 06 (manifest validation), ADR-008, Safety Invariant 5

## Objective

Prove that a tool descriptor with mutually exclusive risk flags (`read_only = true` AND `destructive = true`) or with `read_only = true` AND `open_world = true` AND `requires_interaction = true` against a non-read-only handler is rejected at registration / manifest validation rather than treated as read-only at policy time.

## Preconditions

- Test extension manifest declaring a tool with `read_only = true` and `destructive = true`.
- Native provider attempt registering a descriptor with the same contradiction.

## Test Steps

1. Install the bad extension.
   - **Expected:** Daemon refuses to mark the tool executable; operator view shows `extension_runtime_mismatch` or descriptor-validation reason. Session projection hides the tool.
2. Add a `native_go` provider attempting to register a contradictory descriptor.
   - **Expected:** Registration fails with deterministic descriptor-validation error; daemon log records the violation; tool is absent from operator and session views.
3. Re-emit a valid descriptor and confirm normal registration succeeds.

## Edge Cases

- A tool that legitimately mutates external state but classifies itself `read_only = false`, `destructive = true`, `open_world = true` must register successfully and never auto-approve under `approve-reads`.
- Hooks cannot patch `read_only` flag at dispatch time — descriptor risk classification is install-time authoritative.

## Automation

- **Target:** Unit + Integration
- **Status:** Existing partial coverage in `internal/tools` validation; Missing the cross-backend matrix
- **Command/Spec:** `go test ./internal/tools -run TestDescriptorRiskValidation`
- **Notes:** A mislabeled mutating tool is a Severity = Critical bug because it exposes write authority to `approve-reads` users.
