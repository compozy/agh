## TC-REG-003: Generated Consumers And MVP Web Boundary Stay Truthful

**Priority:** P1
**Type:** Regression
**Status:** Passed
**Estimated Time:** 25 minutes
**Created:** 2026-05-02
**Last Updated:** 2026-05-02

---

### Objective

Verify generated contracts, SDK exports, docs guard tests, and Web guards reflect the runtime MVP without adding unsupported Soul/Heartbeat editors or fake status controls.

---

### Preconditions

- [ ] Repository dependencies are installed.
- [ ] Generated OpenAPI and TypeScript contract files are present.

---

### Test Steps

1. **Check OpenAPI/TypeScript generation drift**
   - Input: `make codegen-check`
   - **Expected:** No generated contract drift.

2. **Run monorepo TypeScript tests that guard authored-context truth**
   - Input: `make bun-test`
   - **Expected:** Web, site, UI, and SDK tests pass, including authored-context guard tests.

3. **Confirm docs/CLI command truth**
   - Input: Inspect generated CLI references for `agh agent soul`, `agh agent heartbeat`, and `agh session health|status|inspect`.
   - **Expected:** Docs include implemented commands and explicitly omit unsupported `agh agent heartbeat refresh` and Web editor promises.

4. **Confirm Web UI boundary**
   - Input: Search Web routes/components for Soul/Heartbeat editor controls.
   - **Expected:** No unsupported editor/status-control UI exists; Web generated types are present for future truthful surfaces only.

---

### Required Evidence

- `qa/evidence/TC-REG-003-codegen-check.log`
- `qa/evidence/TC-REG-003-bun-test.log`
- `qa/evidence/TC-REG-003-web-boundary.log`

---

### Pass Criteria

- Generated consumers and docs remain truthful.
- No Web-only implementation path or fake editor is counted as MVP evidence.
