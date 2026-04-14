## TC-SEC-004: Extension Without task.write Capability Denied Task Creation

**Priority:** P0
**Type:** Security
**Risk Level:** High
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that an authenticated extension principal that lacks the `task.write` capability is denied task creation (and all other write operations). The task system's `requireWriteAuthority` check must enforce that `Authority.Write == true` before allowing any mutation.

---

### Preconditions
- [ ] AGH daemon running with task subsystem and extension host API initialized
- [ ] Extension runtime registered with read-only authority (`Authority{Read: true, Write: false}`)
- [ ] Extension actor context derived via `DeriveExtensionActorContext`

---

### Test Steps
1. **Extension attempts task creation without write capability**
   - Input: Extension calls `CreateTask` with actor context `Authority{Read: true, Write: false, CreateGlobal: false, CreateWorkspace: false}`
   - **Expected:** `ErrPermissionDenied` returned. No task persisted in the store.

2. **Extension attempts task update without write capability**
   - Input: Extension calls `UpdateTask` on an existing task with read-only authority
   - **Expected:** `ErrPermissionDenied` returned. Task unchanged.

3. **Extension attempts task cancellation without write capability**
   - Input: Extension calls `CancelTask` with read-only authority
   - **Expected:** `ErrPermissionDenied` returned. Task status unchanged.

4. **Extension attempts run enqueue without write capability**
   - Input: Extension calls `EnqueueRun` with read-only authority
   - **Expected:** `ErrPermissionDenied` returned. No run created.

5. **Extension with write but without CreateGlobal attempts global task creation**
   - Input: Extension calls `CreateTask` with `Authority{Read: true, Write: true, CreateGlobal: false, CreateWorkspace: true}` and `scope: "global"`
   - **Expected:** `ErrPermissionDenied` returned. Global scope creation blocked by `requireCreateAuthority`.

6. **Extension with write but without CreateWorkspace attempts workspace task creation**
   - Input: Extension calls `CreateTask` with `Authority{Read: true, Write: true, CreateGlobal: true, CreateWorkspace: false}` and `scope: "workspace"`
   - **Expected:** `ErrPermissionDenied` returned. Workspace scope creation blocked.

7. **Extension with full write capability succeeds (control)**
   - Input: Extension calls `CreateTask` with `FullAccessAuthority()`
   - **Expected:** Task created successfully. This confirms the capability gate is the only barrier.

---

### Attack Vectors
- [ ] Extension attempts to bypass capability check by crafting an actor context with elevated authority
- [ ] Extension host API maps `ErrPermissionDenied` to appropriate error code (not leaking internals)
- [ ] Extension attempts write via run lifecycle endpoints (claim, start, complete, fail) without write authority

---

### Related Test Cases
- TC-SEC-003: Unauthenticated request rejection
- TC-SEC-008: Unauthorized scope read for extensions
