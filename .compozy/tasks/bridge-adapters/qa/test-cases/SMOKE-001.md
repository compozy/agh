## SMOKE-001: Bridge SDK Runtime Boots Successfully

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-15

---

### Objective

Verify the shared bridge SDK runtime initializes, completes the JSON-RPC handshake, and reaches a ready state via `bridgesdk.Runtime.Serve()`.

### Preconditions

- [ ] `internal/bridgesdk` package compiles
- [ ] Test harness with mock stdio pipes available

### Test Steps

1. **Create a Runtime with valid config (platform, provider, handlers)**
   - **Expected:** Runtime instance created without error

2. **Start `Serve()` with piped stdin/stdout carrying an InitializeBridgeRuntime payload**
   - **Expected:** Handshake completes, session populated with provider identity and managed instances

3. **Send a health_check request**
   - **Expected:** Health handler invoked, no error returned

4. **Send a shutdown request**
   - **Expected:** Graceful shutdown, Serve() returns nil

### Related Test Cases

- TC-FUNC-002, TC-INT-001
