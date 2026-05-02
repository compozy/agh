## TC-REG-002: CLI And HTTP CAS Contract Parity

**Priority:** P1
**Type:** Integration
**Status:** Passed
**Estimated Time:** 30 minutes
**Created:** 2026-05-02
**Last Updated:** 2026-05-02

---

### Objective

Verify mutating authored-context transports use one body-level CAS contract, and HTTP `If-Match` does not become a second contract.

---

### Preconditions

- [ ] Daemon/API readiness is confirmed.
- [ ] Valid `SOUL.md` and `HEARTBEAT.md` files exist for test agents through managed authoring.
- [ ] Current digests are known from inspect responses.

---

### Test Steps

1. **Use CLI Soul CAS flag**
   - Input: `agh agent soul write reviewer --file <updated-soul> --expected-digest <current-digest> --workspace <workspace> --json`
   - **Expected:** CLI maps `--expected-digest` to body `expected_digest`, mutation succeeds, and inspect shows the new digest.

2. **Use CLI Heartbeat CAS alias**
   - Input: `agh agent heartbeat write ops --file <updated-heartbeat> --if-match <current-digest> --workspace <workspace> --json`
   - **Expected:** CLI maps `--if-match` to body `expected_digest`, mutation succeeds, and inspect/status shows the new digest.

3. **Reject HTTP If-Match-only Soul mutation**
   - Input: `PUT /api/agents/reviewer/soul` with `If-Match: <digest>` and no body `expected_digest`.
   - **Expected:** HTTP rejects the request deterministically and the file remains unchanged.

4. **Reject HTTP If-Match-only Heartbeat mutation**
   - Input: `PUT /api/agents/ops/heartbeat` with `If-Match: <digest>` and no body `expected_digest`.
   - **Expected:** HTTP rejects the request with `heartbeat_if_match_header_unsupported` or equivalent deterministic authored-context diagnostic.

5. **Accept HTTP body expected_digest**
   - Input: `PUT /api/agents/<agent>/<soul-or-heartbeat>` with JSON body containing `expected_digest`.
   - **Expected:** Mutation succeeds and CLI inspect agrees on the new digest.

---

### Required Evidence

- `qa/evidence/TC-REG-002-cli.log`
- `qa/evidence/TC-REG-002-http-if-match.json`
- `qa/evidence/TC-REG-002-http-body-cas.json`

---

### Pass Criteria

- CLI and HTTP agree on body-level CAS semantics.
- HTTP `If-Match` is not accepted as a hidden alternate contract.
- Failed CAS attempts preserve current authored files.
