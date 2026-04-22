## TC-INT-006: Global SQLite Session Index Migrates and Preserves the Provider Column

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 18 minutes
**Created:** 2026-04-21
**Last Updated:** 2026-04-21
**Module:** Global DB migration
**Traceability:** Task 03 migration requirements; ADR-005; TechSpec "Global DB schema" and "Testing Approach"

---

### Objective

Verify that opening legacy global session indexes adds `sessions.provider` safely and preserves provider data on any rebuild path exercised by the repository.

---

### Preconditions

- [ ] A legacy SQLite fixture exists with a `sessions` table that predates the `provider` column.
- [ ] SQLite schema inspection is available.
- [ ] If the repo exposes a copy-style rebuild path, a fixture or command exists to exercise it.

---

### Test Steps

1. Start AGH against the legacy global DB fixture.
   **Expected:** Initialization succeeds and the session index is migrated in place.

2. Inspect the `sessions` table schema.
   **Expected:** The table now contains `provider TEXT NOT NULL DEFAULT ''`.

3. Inspect representative session rows after migration or rebuild.
   **Expected:** Existing rows are preserved, and any rows that already had provider values through the rebuild path keep them intact.

4. Reopen the same DB a second time.
   **Expected:** Migration is idempotent; no duplicate column changes or data loss occur.

---

### Evidence to Capture

- `PRAGMA table_info(sessions)` or equivalent schema evidence.
- Representative `SELECT id, provider FROM sessions` output after migration.
- Any rebuild-path evidence showing provider values survive the copy-style migration.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Legacy DB opened once | No `provider` column initially | Column is added safely. |
| Legacy DB reopened | Already migrated DB | No further schema churn occurs. |
| Rebuild/copy-style path | Existing provider values present | Provider values survive the rebuild. |

---

### Related Test Cases

- `TC-INT-007` for post-migration legacy metadata repair

---

### Notes

Migration is lower priority than the core runtime flows but still required before task 08 can claim full end-to-end coverage.
