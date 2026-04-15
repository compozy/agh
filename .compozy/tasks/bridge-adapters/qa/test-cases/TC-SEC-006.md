## TC-SEC-006: Host API Instance Ownership Authorization

**Priority:** P0
**Type:** Security
**Risk Level:** Critical
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-15

---

### Objective
Verify that the Host API enforces strict instance ownership boundaries, ensuring that extensions (bridge providers) can only access, list, and manage bridge instances they own, preventing cross-provider data leakage and unauthorized instance manipulation.

### Preconditions
- [ ] Bridge adapter runtime is running with at least two registered extensions:
  - Extension A (e.g., Slack provider) with instances `slack-inst-1` and `slack-inst-2`
  - Extension B (e.g., Discord provider) with instances `discord-inst-1`
- [ ] Each extension has its own authentication credentials / context for Host API calls
- [ ] Host API endpoints are accessible: `instances/get`, `instances/list`, `instances/update`, `instances/delete`

### Test Steps

1. **Extension A lists own instances**
   - Input: Extension A calls `instances/list`.
   - **Expected:** Response contains `slack-inst-1` and `slack-inst-2` only. No Discord instances appear in the list.

2. **Extension B lists own instances**
   - Input: Extension B calls `instances/list`.
   - **Expected:** Response contains `discord-inst-1` only. No Slack instances appear.

3. **Extension A gets own instance by ID**
   - Input: Extension A calls `instances/get` with `instance_id: "slack-inst-1"`.
   - **Expected:** 200 OK. Returns full details of `slack-inst-1`.

4. **Extension A attempts to get Extension B's instance**
   - Input: Extension A calls `instances/get` with `instance_id: "discord-inst-1"`.
   - **Expected:** 404 Not Found or 403 Forbidden. Extension A must not see Extension B's instance details. Error response does not confirm or deny the instance exists (to prevent enumeration).

5. **Extension B attempts to get Extension A's instance**
   - Input: Extension B calls `instances/get` with `instance_id: "slack-inst-1"`.
   - **Expected:** 404 Not Found or 403 Forbidden. Symmetric enforcement.

6. **Extension A attempts to update Extension B's instance**
   - Input: Extension A calls `instances/update` with `instance_id: "discord-inst-1"` and modified configuration.
   - **Expected:** 404 or 403. No modification applied to `discord-inst-1`. Extension B's instance remains unchanged.

7. **Extension A attempts to delete Extension B's instance**
   - Input: Extension A calls `instances/delete` with `instance_id: "discord-inst-1"`.
   - **Expected:** 404 or 403. `discord-inst-1` is not deleted. Extension B can still access it.

8. **Instance ID enumeration resistance**
   - Input: Extension A calls `instances/get` with IDs: `discord-inst-1`, `nonexistent-inst`, `linear-inst-99`.
   - **Expected:** All return the same error code (404 or 403). Response timing and error messages are indistinguishable between "exists but not yours" and "does not exist," preventing enumeration.

9. **Cross-provider event delivery attempt**
   - Input: Extension A attempts to deliver an event (via Host API) targeting `discord-inst-1` (owned by Extension B).
   - **Expected:** Rejected. Events can only be delivered to instances owned by the calling extension.

10. **Newly created instance inherits correct ownership**
    - Input: Extension A creates a new instance `slack-inst-3` via Host API.
    - **Expected:** `slack-inst-3` is owned by Extension A. Extension A can list/get it. Extension B cannot see or access it.

11. **No wildcard or admin access from extension context**
    - Input: Extension A calls `instances/list` with a filter like `owner: "*"` or `all: true` (if such parameters exist).
    - **Expected:** Filter is ignored or rejected. Extension A still only sees its own instances. No escalation to admin-level visibility.

### Attack Vectors
- [ ] Horizontal privilege escalation: Extension A accessing Extension B's instances
- [ ] Instance ID guessing/enumeration via timing or error message differences
- [ ] IDOR (Insecure Direct Object Reference) via direct instance_id manipulation
- [ ] Cross-provider event injection by spoofing instance ownership in delivery calls
- [ ] Filter/query parameter manipulation to bypass ownership scoping
- [ ] Race conditions during instance creation/deletion affecting ownership assignment

### Related Test Cases
- TC-SEC-005 (DM policy — complementary access control at the user/sender level)
- TC-SEC-007 (Secret isolation — ensures secrets don't leak across provider boundaries)
- TC-SEC-010 (Config injection — prevents cross-contamination of provider configurations)
