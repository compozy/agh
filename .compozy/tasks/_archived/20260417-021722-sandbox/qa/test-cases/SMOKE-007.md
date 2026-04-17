## SMOKE-007: Environment hooks fire during session lifecycle

**Priority:** P0 (Critical)
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16

---

### Objective

Verify all 5 environment hook events are dispatched at the correct lifecycle points during a session create-stop cycle.

---

### Preconditions

- [x] Hook events registered in `allHookEvents`
- [x] Session manager dispatches hooks from environment lifecycle
- [x] Test hook observer or native hook registered

---

### Test Steps

1. **Create session with local environment and registered hook observer**
   - **Expected:** `environment.prepare` fires before `Provider.Prepare()` with `workspace_id` and `backend` in payload

2. **Verify environment.ready fires**
   - **Expected:** `environment.ready` fires after Prepare succeeds with `instance_id` and `runtime_root` in payload

3. **Stop session**
   - **Expected:** `environment.sync.before` fires before sync, `environment.sync.after` fires after sync with stats, `environment.stop` fires before teardown

4. **Verify event order**
   - **Expected:** prepare -> ready -> (session active) -> sync.before -> sync.after -> stop
