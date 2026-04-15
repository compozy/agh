## TC-SEC-005: DM Policy Enforcement

**Priority:** P0
**Type:** Security
**Risk Level:** Critical
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-04-15

---

### Objective

Verify that DM (Direct Message) policy enforcement correctly controls which external users can interact with bridge instances, ensuring that the `open`, `allowlist`, and `pairing` policies are applied accurately and that unauthorized users receive clear rejection responses.

### Preconditions

- [ ] Bridge adapter runtime is running with three separate bridge instances configured:
  - Instance A: DM policy set to `open`
  - Instance B: DM policy set to `allowlist` with pre-approved user IDs (e.g., `["U001", "U002", "user@example.com"]`)
  - Instance C: DM policy set to `pairing` with no paired users initially
- [ ] A mechanism to pair users to Instance C is available (e.g., Host API call or CLI command)
- [ ] Webhook requests can be crafted with different sender/peer identifiers
- [ ] All requests use valid signatures (signature is not the variable under test)

### Test Steps

1. **Open policy — any sender accepted**
   - Input: Send a valid webhook to Instance A (open policy) with `peer_id: "U999"` (an arbitrary, unknown user).
   - **Expected:** Request accepted. Message delivered to the bridge instance. No sender filtering applied.

2. **Open policy — multiple distinct senders**
   - Input: Send 5 webhooks to Instance A, each from a different `peer_id` (`U100`, `U200`, `U300`, `U400`, `U500`).
   - **Expected:** All 5 requests accepted and processed. Open policy imposes no restrictions on sender identity.

3. **Allowlist policy — approved user accepted**
   - Input: Send a valid webhook to Instance B (allowlist policy) with `peer_id: "U001"` (a pre-approved user).
   - **Expected:** Request accepted. Message delivered to the bridge instance.

4. **Allowlist policy — second approved user accepted**
   - Input: Send a valid webhook to Instance B with `peer_id: "U002"`.
   - **Expected:** Request accepted.

5. **Allowlist policy — unapproved user rejected**
   - Input: Send a valid webhook to Instance B with `peer_id: "U999"` (not in the allowlist).
   - **Expected:** Request rejected with 403 Forbidden. Response indicates the user is not authorized. Message is NOT silently dropped — the sender receives a clear rejection.

6. **Allowlist policy — empty peer_id rejected**
   - Input: Send a valid webhook to Instance B with `peer_id: ""` or missing peer_id field.
   - **Expected:** Request rejected. Empty or missing sender identity does not bypass the allowlist.

7. **Allowlist policy — peer_id with different casing**
   - Input: If allowlist contains `"U001"`, send a webhook with `peer_id: "u001"` (lowercase).
   - **Expected:** Behavior depends on implementation: either case-insensitive match (accepted) or case-sensitive match (rejected). Document the actual behavior. No undefined behavior or crash.

8. **Pairing policy — unpaired user rejected**
   - Input: Send a valid webhook to Instance C (pairing policy) with `peer_id: "U001"` before any pairing has occurred.
   - **Expected:** Request rejected with 403 Forbidden. Clear error message indicates the user has not been paired.

9. **Pairing policy — pair a user, then send message**
   - Input: (a) Pair `peer_id: "U001"` to Instance C via the pairing mechanism. (b) Send a valid webhook to Instance C with `peer_id: "U001"`.
   - **Expected:** (a) Pairing succeeds. (b) Request accepted. Message delivered to the bridge instance.

10. **Pairing policy — paired user accepted, unpaired user still rejected**
    - Input: After pairing `U001` to Instance C, send a webhook with `peer_id: "U002"` (not paired).
    - **Expected:** Request rejected with 403. Pairing is per-user, not a global unlock.

11. **Pairing policy — unpair a user, then send message**
    - Input: (a) Unpair `peer_id: "U001"` from Instance C. (b) Send a valid webhook with `peer_id: "U001"`.
    - **Expected:** (a) Unpairing succeeds. (b) Request rejected with 403. Pairing revocation is immediate.

12. **Policy enforcement does not leak user list**
    - Input: Send a webhook to Instance B (allowlist) with an unapproved user. Inspect the error response body.
    - **Expected:** Response does not contain the list of approved users. Error message is generic (e.g., "user not authorized") without revealing who IS authorized.

13. **Policy change at runtime**
    - Input: Change Instance A's policy from `open` to `allowlist` with an empty allowlist (if hot-reconfiguration is supported). Send a webhook with any `peer_id`.
    - **Expected:** If hot-reconfiguration is supported: request rejected (empty allowlist blocks everyone). If not: behavior is defined and documented (e.g., requires instance restart).

### Attack Vectors

- [ ] Unauthorized user sending DMs to a restricted bridge instance
- [ ] Empty or missing peer_id bypassing identity checks
- [ ] Case sensitivity exploits in allowlist matching
- [ ] Enumeration of allowlisted users via error message differences
- [ ] Race condition between pairing/unpairing and message delivery
- [ ] Policy confusion by switching policies at runtime without proper state reset
- [ ] Peer_id spoofing (mitigated by signature verification but tested here at the policy layer)

### Related Test Cases

- TC-SEC-001 (Signature verification — authenticates the webhook source before DM policy is applied)
- TC-SEC-006 (Host API authorization — controls which extensions can manage instances and policies)
- TC-SEC-007 (Secret isolation — ensures policy configuration is not leaked)
