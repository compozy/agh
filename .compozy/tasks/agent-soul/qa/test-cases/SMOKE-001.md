## SMOKE-001: Repository And Runtime Readiness

**Priority:** P0
**Type:** Smoke
**Status:** Passed
**Estimated Time:** 20 minutes
**Created:** 2026-05-02
**Last Updated:** 2026-05-02

---

### Objective

Verify the repository, generated contracts, and isolated AGH runtime are healthy enough to execute behavior-first Agent Soul/Heartbeat QA.

---

### Preconditions

- [ ] Root instructions and PRD docs have been read.
- [ ] QA bootstrap helper is available.
- [ ] The workflow QA output path is `.compozy/tasks/agent-soul`.

---

### Test Steps

1. **Discover the project QA contract**
   - Input: `python3 .agents/skills/qa-execution/scripts/discover-project-contract.py --root .`
   - **Expected:** The canonical gate is identified as `make verify`, with Go and Bun surfaces detected.

2. **Run generated contract readiness checks**
   - Input: `make codegen-check`
   - **Expected:** The generated OpenAPI and TypeScript contract consumers have no drift.

3. **Create a fresh isolated QA lab**
   - Input: `python3 .agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py --scenario agent-soul --repo-root .`
   - **Expected:** The helper writes `qa/bootstrap-manifest.json` and `qa/bootstrap.env` with unique `AGH_HOME`, HTTP port, UDS path, provider home, and browser policy.

4. **Start or confirm daemon readiness in the isolated lab**
   - Input: Use the bootstrap env, then run the supported daemon start/status commands.
   - **Expected:** Daemon health is observable through CLI status and HTTP readiness.

5. **Confirm no MVP Web editor is required**
   - Input: Inspect generated Web/Site guard tests and docs references for Soul/Heartbeat.
   - **Expected:** Generated contracts exist, but no unsupported Soul/Heartbeat Web editor is required for MVP QA.

---

### Required Evidence

- `qa/evidence/project-contract.json`
- `qa/bootstrap-manifest.json`
- `qa/evidence/smoke-codegen-check.log`
- `qa/evidence/smoke-daemon-status.log`

---

### Pass Criteria

- The isolated lab exists and daemon readiness is proven.
- Contract drift checks pass or a blocking bug is filed.
- Smoke evidence is clearly marked readiness-only in the final report.
