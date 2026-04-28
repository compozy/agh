## TC-SEC-002: Env var allowlist blocks DAYTONA_API_KEY

**Priority:** P0 (Critical)
**Type:** Security
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 06
**Risk Level:** High

---

### Objective

Verify that the environment variable allowlist in `internal/sandbox/daytona/env.go` prevents `DAYTONA_API_KEY`, `DAYTONA_JWT_TOKEN`, and other daemon-internal secrets from being propagated to remote sandboxes.

---

### Test Steps

1. **Filter env vars through allowlist**
   - Input: Env containing `DAYTONA_API_KEY=secret`, `AGH_SESSION_ID=sess-123`, `HOME=/home/user`
   - **Expected:** `DAYTONA_API_KEY` removed, `AGH_SESSION_ID` kept

2. **Verify DAYTONA_JWT_TOKEN blocked**
   - Input: `DAYTONA_JWT_TOKEN=jwt-abc`
   - **Expected:** Blocked from propagation

3. **Verify daemon-internal vars blocked**
   - Input: Various internal daemon vars
   - **Expected:** None leak to sandbox

---

### Compliance Check

- [x] No daemon secrets propagated to sandbox
- [x] Only `AGH_*` session vars and user-declared vars propagated
- [x] Profile-level `Env` overrides included
