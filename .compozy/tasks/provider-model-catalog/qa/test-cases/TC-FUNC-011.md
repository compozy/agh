# TC-FUNC-011: Extension `model.source` Manifest + Row Validation

**Priority:** P1
**Type:** Functional
**Module:** `internal/extension`
**Requirement:** ADR-003, TechSpec Extension Sources.
**Status:** Not Run

## Objective

Verify extension manifests can declare a `model.source` capability with a normalizable slug; non-normalizable slugs are rejected; `models/list` results pass through `internal/modelcatalog` validation; invalid rows are dropped with deterministic source-status errors.

## Preconditions

- [ ] Extension fixture with manifest declaring `model.source` capability for one provider.
- [ ] Daemon configured to register that extension.

## Test Steps

1. **Manifest accepts normalizable slug.**
   - Manifest declares `name = "Acme Models"` mapped to slug `acme-models`.
   - **Expected:** Daemon registers `source_id="extension:acme-models"`; manifest validation passes.
2. **Manifest rejects unmappable slug.**
   - Manifest declares `name = "??"`.
   - **Expected:** Validation fails with deterministic error referencing the manifest field.
3. **Extension returns valid rows.**
   - `models/list` returns rows with provider/model IDs the extension declares.
   - **Expected:** Rows persist; merge applies extension priority 100; status `succeeded`.
4. **Extension returns invalid rows.**
   - Stub returns row with empty `model_id`.
   - **Expected:** Row rejected; remaining valid rows persist; source status records redacted error referencing the offending field.
5. **Extension declares provider it has no grant for.**
   - **Expected:** Source status reports `failed` with capability-missing error; no rows persisted.

## Audit Coverage

- C6 task tree (Task 08).
- SI-8 (only `internal/modelcatalog.Store` writes rows), SI-9 (redaction).

## Pass Criteria

- Manifest validation matches Task 08 fixtures.
- Invalid rows do not pollute persisted catalog.
- Capability gate enforced.

## Failure Criteria

- Invalid manifest passes validation.
- Invalid row corrupts persisted state.
- Capability gate bypassed.
