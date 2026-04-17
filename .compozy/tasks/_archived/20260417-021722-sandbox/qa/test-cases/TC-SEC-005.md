## TC-SEC-005: Network policy unsupported setting logs warning

**Priority:** P2 (Medium)
**Type:** Security
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 06
**Risk Level:** Medium

---

### Objective

Verify network policy enforcement behavior: `AllowPublicIngress` is enforced, unsupported policies (AllowOutbound, AllowList, DenyList) log warnings, and required unsupported policies return errors.

---

### Test Steps

1. **AllowPublicIngress=false**
   - **Expected:** Daytona sandbox preview links configured as private

2. **AllowOutbound=false (unsupported by Daytona)**
   - Input: `AllowOutbound = false`, `Required = false`
   - **Expected:** Warning logged, sandbox still created

3. **Required unsupported policy**
   - Input: `AllowOutbound = false`, `Required = true`
   - **Expected:** `Provider.Prepare()` returns error indicating policy cannot be enforced

4. **DenyList with entries**
   - Input: `DenyList = ["10.0.0.0/8"]`, `Required = false`
   - **Expected:** Warning logged, policy not enforced
