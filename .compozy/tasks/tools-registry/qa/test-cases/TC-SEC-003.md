# TC-SEC-003 — `approve-all` does not bypass explicit denies, lineage, or hooks

- **Priority:** P0
- **Type:** Security / policy
- **Trace:** Task 03, Task 04, ADR-005, Safety Invariant 4

## Objective

Prove that `approve-all` only removes the auto-prompt for *otherwise-allowed* tools. It must not bypass `deny_tools`, conflicted state, source denies, agent allow lists, session lineage subsetting, hook denial, or unavailable backends.

## Preconditions

- `permissions.mode = "approve-all"`.
- Agent definition with `deny_tools = ["agh__network_send"]`.
- Pre-call hook configured to deny one specific `tool_id` (`agh__task_create`) returning `hook_denied`.
- One `conflicted` tool (collision intentionally created via two providers).
- Child session whose lineage atoms exclude `agh__skill_view`.
- Source policy denies `extension:bad_ext`.

## Test Steps

1. Invoke `agh__network_send` from agent context.
   - **Expected:** Denied with `policy_denied`, denying layer `agent_policy`. No prompt.
2. Invoke `agh__task_create`.
   - **Expected:** Pre-call hook fires, denies, reason `hook_denied`.
3. Invoke the conflicted tool ID.
   - **Expected:** Returns `tool_conflict` (HTTP 409 / CLI error code `conflicted_id`).
4. Invoke `agh__skill_view` from child session.
   - **Expected:** Denied with `session_denied`, layer `session_lineage`.
5. Invoke any tool from `extension:bad_ext`.
   - **Expected:** Denied with `policy_denied`, layer `source_policy`.
6. Invoke a normally allowed `agh__skill_view` from a parent session.
   - **Expected:** Auto-approved, returns content.

## Edge Cases

- Approval bridge MUST NOT attempt ACP `session/request_permission` for `approve-all` allowed paths.
- `approve-all` must NOT raise authority above an unavailable backend (e.g. unhealthy extension).

## Automation

- **Target:** Integration
- **Status:** Existing for individual denies; Missing for the combined matrix
- **Command/Spec:** `go test ./internal/tools -run TestPolicyApproveAllMatrix`
- **Notes:** Common misunderstanding that `approve-all` means "execute everything"; this case enforces docs reality.
