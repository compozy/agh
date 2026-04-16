## TC-INT-002: Full local session lifecycle through daemon

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 3 minutes
**Created:** 2026-04-16
**Tasks:** 03, 04

---

### Objective

Verify the complete session lifecycle (create -> prompt -> file read/write -> terminal -> stop) works end-to-end through the daemon with the local provider, confirming the environment abstraction layer is transparent for local execution.

---

### Preconditions

- [x] Daemon wiring complete with environment registry
- [x] Local provider registered
- [x] Mock ACP server subprocess available

---

### Test Steps

1. **Create session with default (local) environment**
   - **Expected:** Session created, environment metadata shows `backend: local`, `state: prepared`

2. **Send prompt via ACP driver**
   - **Expected:** Agent responds, ACP protocol works through local Launcher

3. **Read file via ACP fs/read**
   - **Expected:** File content returned via localToolHost.ReadTextFile

4. **Write file via ACP fs/write**
   - **Expected:** File created via localToolHost.WriteTextFile

5. **Create terminal**
   - **Expected:** Terminal spawned via localToolHost.CreateTerminal

6. **Stop session**
   - **Expected:** Session stops, SyncFromRuntime no-op, Destroy no-op, session metadata shows stopped state

---

### Error Scenarios

- [x] Network timeout handling: N/A for local
- [x] Invalid response handling: ACP error propagation works
- [x] Authentication failure: N/A for local
