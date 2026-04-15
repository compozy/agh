## SMOKE-006: Error Classification Maps Provider Failures

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-15

---

### Objective

Verify the SDK error classifier correctly maps representative HTTP errors to the 5 error classes.

### Preconditions

- [ ] `internal/bridgesdk` errors package available

### Test Steps

1. **Classify an HTTP 401 error**
   - **Expected:** Classified as `ErrorClassAuth`

2. **Classify an HTTP 429 error**
   - **Expected:** Classified as `ErrorClassRateLimit`

3. **Classify an HTTP 500 error**
   - **Expected:** Classified as `ErrorClassTransient`

4. **Classify an HTTP 404 error**
   - **Expected:** Classified as `ErrorClassPermanent`

5. **Classify a context.DeadlineExceeded error**
   - **Expected:** Classified as `ErrorClassTimeout`

### Related Test Cases

- TC-FUNC-013, TC-FUNC-014
