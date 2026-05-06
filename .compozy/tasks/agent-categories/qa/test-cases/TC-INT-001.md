## TC-INT-001: API / Contract / Bundle / Native-Tool Propagate `category_path`

**Priority:** P0
**Type:** Integration
**Module:** `internal/api/contract` + `internal/api/core` + `internal/bundles` + `internal/daemon` native tools
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Verify that `category_path` propagates through every conversion seam without loss:

- `AgentPayloadFromDef` copies the field defensively (mutating the source after conversion does not leak).
- `AgentPayloadFromDiagnostic` keeps the field nil because diagnostics describe malformed files.
- `BundleAgentPayload` carries the field on activation payloads.
- HTTP `/api/agents`, `/api/agents/:name`, `/api/workspaces/:id`, and the bundle activation endpoint expose the field.
- The UDS counterparts return the same payload.
- The native tool `agh__workspace_describe` returns `category_path` on each agent.

---

### Preconditions

- [ ] Daemon is up against an isolated `AGH_HOME` with at least one categorized agent, one root-level agent, one bundle activation that ships a categorized agent, and one diagnostic-producing AGENT.md.
- [ ] HTTP base URL and UDS socket from bootstrap manifest are available.

---

### Test Steps

1. **Defensive copy in `AgentPayloadFromDef`.**
   - Input: Build an `AgentDef` with `CategoryPath = []string{"Marketing", "Sales"}`. Convert to payload. Mutate the source slice (e.g., `def.CategoryPath[0] = "Hacked"`).
   - **Expected:** `payload.CategoryPath` still equals `["Marketing", "Sales"]`.

2. **Diagnostic exclusion.**
   - Input: A malformed AGENT.md that surfaces via `AgentPayloadFromDiagnostic`.
   - **Expected:** `payload.CategoryPath` is nil. The diagnostic record itself describes the validation error (e.g., blank segment), but the placeholder agent does not surface a fabricated category.

3. **HTTP `/api/agents` parity.**
   - Input: `curl -sS $BASE/api/agents | jq '.agents[] | {name, category_path}'`
   - **Expected:** Categorized agent has `category_path: ["Marketing", "Sales"]`. Root-level agent omits the field.

4. **HTTP `/api/agents/:name` parity.**
   - Input: `curl -sS $BASE/api/agents/<categorized-agent> | jq '.agent.category_path'`
   - **Expected:** `["Marketing", "Sales"]`.

5. **HTTP `/api/workspaces/:id` parity.**
   - Input: `curl -sS $BASE/api/workspaces/<id> | jq '.agents[] | {name, category_path}'`
   - **Expected:** Same shape as step 3.

6. **Bundle activation parity.**
   - Input: Hit the bundle activation endpoint or call the bundle projector test helper. Inspect the agent payloads.
   - **Expected:** Each shipped agent carries `category_path` matching its AGENT.md.

7. **UDS parity.**
   - Input: `agh agent list -o json` (UDS-backed) and the underlying `agh agent info` UDS call.
   - **Expected:** `category_path` array matches HTTP exactly.

8. **Native tool parity.**
   - Input: Call `agh__workspace_describe` (e.g., via a session that requests it, or via the daemon-tool test harness) and inspect the agent payloads.
   - **Expected:** `category_path` is present on each agent and equals the HTTP/UDS values.

---

### Behavioral Evidence

- Cross-surface: HTTP, UDS, native tool, and bundle activation agree on the same `category_path` for the same `agent.name`.
- Artifact reuse: bundle payload is later re-inflated and still carries the field.
- Disruption probe: source mutation after conversion proves defensive copy.

---

### Audit Coverage

- C4: daemon, agents, observer.
- C5: HTTP, UDS, native tools, bundle activation.
- C8: four-way cross-surface truth.
- C10: bundle payload reuse.
- C11: source-mutation disruption.
- C14: `go test ./internal/api/...` plus `go test ./internal/bundles/...` plus `go test ./internal/daemon/...`.

---

### Pass Criteria

- All eight steps pass.
- A defensive copy is verifiable.
- Diagnostic agents emit `category_path: null` (or omit it) intentionally.

---

### Failure Criteria

- Any seam silently drops the field.
- Source mutation propagates into the payload.
- Diagnostic placeholder ever exposes a non-nil category.
