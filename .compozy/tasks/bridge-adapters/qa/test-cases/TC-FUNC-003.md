## TC-FUNC-003: Bridge Instance Update

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective
Validate that an existing bridge instance's display_name, provider_config, delivery_defaults, and DM policy can be updated independently, that changes persist correctly, and that old values are fully replaced.

### Preconditions
- [ ] Daemon is running with a bridge provider registered
- [ ] A bridge instance exists with known initial values:
  - `display_name: "Original Name"`
  - `provider_config: {"mode": "bot", "webhook_url": "https://old.example.com"}`
  - `delivery_defaults: {"peer_id": "old-peer", "mode": "direct-send"}`
  - `dm_policy: "open"`
  - `status: "disabled"`, `enabled: false`

### Test Steps
1. **Update display_name only**
   - Input: Update with `display_name: "Updated Bridge Name"`
   - **Expected:**
     - `display_name` = `"Updated Bridge Name"`
     - `provider_config` unchanged (still contains `mode: "bot"`)
     - `delivery_defaults` unchanged (still contains `peer_id: "old-peer"`)
     - `dm_policy` unchanged (`"open"`)
     - `updated_at` timestamp is newer than before the update

2. **Update provider_config with entirely new payload**
   - Input: Update with `provider_config: {"mode": "app", "enterprise_url": "https://corp.api.example.com"}`
   - **Expected:**
     - `provider_config` is exactly `{"mode":"app","enterprise_url":"https://corp.api.example.com"}`
     - Old keys (`webhook_url`) are absent -- full replacement, not merge
     - `delivery_defaults` unchanged
     - `display_name` unchanged

3. **Update delivery_defaults with new values**
   - Input: Update with `delivery_defaults: {"peer_id": "new-peer", "thread_id": "new-thread", "mode": "reply"}`
   - **Expected:**
     - `delivery_defaults` is exactly `{"peer_id":"new-peer","thread_id":"new-thread","mode":"reply"}`
     - Old value (`mode: "direct-send"`) is replaced
     - `provider_config` unchanged

4. **Update DM policy from open to allowlist**
   - Input: Update with `dm_policy: "allowlist"`
   - **Expected:**
     - `dm_policy` = `"allowlist"`
     - All other fields unchanged

5. **Update DM policy to pairing**
   - Input: Update with `dm_policy: "pairing"`
   - **Expected:** `dm_policy` = `"pairing"`

6. **Update routing_policy**
   - Input: Update with `routing_policy: {"include_peer": true, "include_thread": true, "include_group": false}`
   - **Expected:**
     - `routing_policy.include_peer` = `true`
     - `routing_policy.include_thread` = `true`
     - `routing_policy.include_group` = `false`

7. **Clear provider_config by setting null/empty**
   - Input: Update with `provider_config: null`
   - **Expected:** `provider_config` is null/empty in the persisted instance

8. **Verify managed (package-sourced) instance rejects direct update**
   - Input: Create instance with `source: "package"`, then attempt to update `display_name`
   - **Expected:** Rejected with `ErrBridgeInstanceReadOnly`

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Update non-existent instance | Random unknown ID | Error: `ErrBridgeInstanceNotFound` |
| Update with invalid JSON provider_config | `provider_config: "{"` | Validation error: must be valid JSON |
| Update delivery_defaults with invalid mode | `delivery_defaults: {"mode": "broadcast"}` | Validation error: unsupported delivery target mode |
| Concurrent updates | Two simultaneous updates to same instance | Last writer wins; both complete without error; final state is deterministic |
| Update with routing_policy thread without peer/group | `routing_policy: {"include_thread": true}` | Validation error: "routing policy cannot include thread without peer or group" |
| Whitespace-padded display_name | `display_name: "  Padded  "` | Normalized to `"Padded"` |

### Related Test Cases
- TC-FUNC-001 (creation)
- TC-FUNC-006 (provider_config vs delivery_defaults separation)
- TC-FUNC-018 (source distinction)
