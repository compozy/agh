## SMOKE-007: Sandbox hooks fire during session lifecycle

**Priority:** P0 (Critical)
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16

---

### Objective

Verify all 5 sandbox hook events are dispatched at the correct lifecycle points during a session create-stop cycle.

---

### Preconditions

- [x] Hook events registered in `allHookEvents`
- [x] Session manager dispatches hooks from sandbox lifecycle
- [x] Test hook observer or native hook registered

---

### Test Steps

1. **Create session with local sandbox and registered hook observer**
   - **Expected:** `sandbox.prepare` fires before `Provider.Prepare()` with `workspace_id` and `backend` in payload

2. **Verify sandbox.ready fires**
   - **Expected:** `sandbox.ready` fires after Prepare succeeds with `instance_id` and `runtime_root` in payload

3. **Stop session**
   - **Expected:** `sandbox.sync.before` fires before sync, `sandbox.sync.after` fires after sync with stats, `sandbox.stop` fires before teardown

4. **Verify event order**
   - **Expected:** prepare -> ready -> (session active) -> sync.before -> sync.after -> stop
