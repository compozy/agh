## TC-FUNC-017: Delivery Target Resolution

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-15

---

### Objective
Validate that `BuildDeliveryTarget` correctly merges a bridge instance's `delivery_defaults` with per-delivery request overrides to produce a canonical `DeliveryTarget`, with request values taking precedence over defaults and missing values falling through to defaults.

### Preconditions
- [ ] `internal/bridges` package is compiled and testable
- [ ] `BuildDeliveryTarget` function is available
- [ ] A bridge instance with delivery_defaults:
  ```json
  {"peer_id": "default-peer", "thread_id": "default-thread", "group_id": "default-group", "mode": "direct-send"}
  ```

### Test Steps
1. **Full override: request provides all fields**
   - Input:
     - Instance delivery_defaults: `{"peer_id": "default-peer", "mode": "direct-send"}`
     - Request: `{bridge_instance_id: "inst-001", peer_id: "req-peer", thread_id: "req-thread", group_id: "req-group", mode: "reply"}`
   - **Expected:**
     - `target.PeerID` = `"req-peer"` (request overrides default)
     - `target.ThreadID` = `"req-thread"` (request value)
     - `target.GroupID` = `"req-group"` (request value)
     - `target.Mode` = `"reply"` (request overrides default)
     - `target.BridgeInstanceID` = `"inst-001"`

2. **Partial override: request provides only peer_id**
   - Input:
     - Instance delivery_defaults: `{"peer_id": "default-peer", "thread_id": "default-thread", "mode": "reply"}`
     - Request: `{bridge_instance_id: "inst-001", peer_id: "override-peer"}`
   - **Expected:**
     - `target.PeerID` = `"override-peer"` (overridden)
     - `target.ThreadID` = `"default-thread"` (from defaults)
     - `target.Mode` = `"reply"` (from defaults)

3. **No override: request has no routing fields**
   - Input:
     - Instance delivery_defaults: `{"peer_id": "default-peer", "mode": "direct-send"}`
     - Request: `{bridge_instance_id: "inst-001"}` (no overrides)
   - **Expected:**
     - `target.PeerID` = `"default-peer"` (from defaults)
     - `target.Mode` = `"direct-send"` (from defaults)

4. **Empty defaults, request provides all**
   - Input:
     - Instance delivery_defaults: `null`
     - Request: `{bridge_instance_id: "inst-001", peer_id: "p1", mode: "reply"}`
   - **Expected:**
     - `target.PeerID` = `"p1"`
     - `target.Mode` = `"reply"`

5. **Empty defaults and empty request: mode falls to direct-send**
   - Input:
     - Instance delivery_defaults: `null`
     - Request: `{bridge_instance_id: "inst-001", peer_id: "p1"}` (no mode)
   - **Expected:**
     - `target.Mode` = `"direct-send"` (the hardcoded fallback)

6. **Mode normalization aliases**
   - Input: Request with `mode: "direct"` (alias)
   - **Expected:** Normalized to `"direct-send"`
   - Input: Request with `mode: "reply_send"` (alias)
   - **Expected:** Normalized to `"reply"`

7. **Mismatched bridge_instance_id rejected**
   - Input: Instance ID is `"inst-001"`, request has `bridge_instance_id: "inst-999"`
   - **Expected:** Error: "delivery target request bridge instance id does not match instance"

8. **Validation of resolved target**
   - Input: Defaults and request both empty, resulting in no peer_id or group_id with mode=direct-send
   - **Expected:** Validation error: "delivery target mode direct-send requires peer id or group id"

9. **Thread without peer or group**
   - Input: Resolved target has `thread_id: "t1"` but neither `peer_id` nor `group_id`
   - **Expected:** Validation error: "delivery target thread id requires peer id or group id"

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Defaults with invalid mode | `delivery_defaults: {"mode": "broadcast"}` | Error from `decodeDeliveryTargetDefaults`: unsupported delivery target mode |
| Defaults with invalid JSON | `delivery_defaults: "{bad"` | Error: must be valid JSON |
| Request bridge_instance_id empty | `bridge_instance_id: ""` | Validation error: "delivery target request bridge instance id is required" |
| Whitespace in default peer_id | `{"peer_id": "  padded  "}` | Normalized to `"padded"` |
| Empty object defaults `{}` | `delivery_defaults: {}` | All fields empty, fallback to request values |

### Related Test Cases
- TC-FUNC-006 (delivery_defaults vs provider_config)
- TC-FUNC-009 (delivery events carry resolved target)
- TC-FUNC-016 (routing key construction)
