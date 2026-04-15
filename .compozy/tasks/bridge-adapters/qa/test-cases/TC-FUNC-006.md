## TC-FUNC-006: Provider Config vs Delivery Defaults Separation

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-15

---

### Objective
Verify that `provider_config` and `delivery_defaults` remain distinct JSON payloads on a bridge instance. Updating one must never affect the other. The two payloads serve different purposes: `provider_config` holds provider-specific operational settings, while `delivery_defaults` holds only delivery-target defaults (`peer_id`, `thread_id`, `group_id`, `mode`).

### Preconditions
- [ ] A bridge instance exists with both payloads populated:
  - `provider_config: {"mode": "bot", "webhook_url": "https://hook.example.com", "batch_window_ms": 500}`
  - `delivery_defaults: {"peer_id": "default-peer", "thread_id": "default-thread", "mode": "direct-send"}`

### Test Steps
1. **Update provider_config only**
   - Input: Update with `provider_config: {"mode": "app", "enterprise_url": "https://corp.example.com"}`
   - **Expected:**
     - `provider_config` = `{"mode":"app","enterprise_url":"https://corp.example.com"}`
     - `delivery_defaults` unchanged: `{"peer_id":"default-peer","thread_id":"default-thread","mode":"direct-send"}`

2. **Update delivery_defaults only**
   - Input: Update with `delivery_defaults: {"group_id": "channel-123", "mode": "reply"}`
   - **Expected:**
     - `delivery_defaults` = `{"group_id":"channel-123","mode":"reply"}`
     - `provider_config` unchanged from step 1

3. **Clear provider_config, keep delivery_defaults**
   - Input: Update with `provider_config: null`
   - **Expected:**
     - `provider_config` is null/empty
     - `delivery_defaults` unchanged from step 2

4. **Clear delivery_defaults, keep provider_config**
   - Input: Set `provider_config: {"mode": "bot"}`, then update with `delivery_defaults: null`
   - **Expected:**
     - `delivery_defaults` is null/empty
     - `provider_config` = `{"mode":"bot"}`

5. **Verify delivery_defaults rejects provider-config-style keys**
   - Input: `delivery_defaults: {"peer_id": "p1", "mode": "direct-send", "webhook_url": "https://bad.example.com"}`
   - **Expected:** The `BridgeDeliveryDefaultsPayload` UnmarshalJSON rejects keys outside the approved set (`peer_id`, `thread_id`, `group_id`, `mode`) if strict validation is applied, or the extra keys are silently ignored during target resolution

6. **Verify provider_config accepts arbitrary keys**
   - Input: `provider_config: {"custom_field": "value", "nested": {"deep": true}}`
   - **Expected:** Accepted and persisted as-is (provider_config is opaque JSON)

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Both payloads set to empty objects | `provider_config: {}, delivery_defaults: {}` | Both persist as `{}` or normalized to null |
| Both payloads null simultaneously | `provider_config: null, delivery_defaults: null` | Both cleared |
| Large provider_config (10KB) | 10KB JSON object | Accepted if within body size limits |
| delivery_defaults with invalid mode | `delivery_defaults: {"mode": "broadcast"}` | Validation error: unsupported delivery target mode |

### Related Test Cases
- TC-FUNC-001 (creation with both payloads)
- TC-FUNC-003 (update mechanics)
- TC-FUNC-017 (delivery target resolution uses delivery_defaults)
