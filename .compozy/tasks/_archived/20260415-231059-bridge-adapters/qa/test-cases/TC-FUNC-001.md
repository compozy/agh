## TC-FUNC-001: Bridge Instance Creation

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective

Validate that a bridge instance can be created with valid platform, extension name, scope, and persists with the correct initial status, provider_config, delivery_defaults, and DM policy defaults.

### Preconditions

- [ ] Daemon is running with at least one bridge provider extension registered (e.g., `telegram`)
- [ ] `make build` succeeds
- [ ] SQLite store is available via `t.TempDir()` isolation
- [ ] No pre-existing bridge instances in the test database

### Test Steps

1. **Submit a CreateBridgeRequest with all required fields**
   - Input:
     ```json
     {
       "scope": "global",
       "platform": "telegram",
       "extension_name": "bridges/telegram",
       "display_name": "My Telegram Bridge",
       "enabled": false,
       "status": "disabled",
       "dm_policy": "open",
       "routing_policy": { "include_peer": true, "include_thread": false, "include_group": false },
       "provider_config": { "mode": "bot", "webhook_url": "https://example.com/webhook" },
       "delivery_defaults": { "peer_id": "default-peer", "mode": "direct-send" }
     }
     ```
   - **Expected:** HTTP 201 or successful RPC response with a generated bridge instance ID

2. **Retrieve the created instance by ID**
   - Input: GET `instances/get` with the returned ID
   - **Expected:**
     - `id` is a non-empty string (UUID or similar)
     - `scope` = `"global"`
     - `workspace_id` = `""` (empty for global scope)
     - `platform` = `"telegram"`
     - `extension_name` = `"bridges/telegram"`
     - `display_name` = `"My Telegram Bridge"`
     - `source` = `"dynamic"` (default for operator-created instances)
     - `enabled` = `false`
     - `status` = `"disabled"`
     - `dm_policy` = `"open"`
     - `routing_policy.include_peer` = `true`
     - `provider_config` is valid JSON matching the input object
     - `delivery_defaults` is valid JSON matching the input object
     - `degradation` is `null`
     - `created_at` is a valid RFC3339 timestamp
     - `updated_at` is a valid RFC3339 timestamp >= `created_at`

3. **Verify workspace-scoped creation**
   - Input: Same request but with `scope: "workspace"`, `workspace_id: "ws-001"`
   - **Expected:** Instance persists with `scope=workspace`, `workspace_id=ws-001`

4. **Verify default DM policy when omitted**
   - Input: Same request but omit `dm_policy` field
   - **Expected:** Instance persists with `dm_policy=open` (the normalize default)

5. **Verify default source when omitted**
   - Input: Same request, no `source` field
   - **Expected:** Instance persists with `source=dynamic`

### Edge Cases & Variations

| Variation                            | Input                                     | Expected Result                                                  |
| ------------------------------------ | ----------------------------------------- | ---------------------------------------------------------------- |
| Missing platform                     | `platform: ""`                            | Validation error: "bridge instance platform is required"         |
| Missing extension_name               | `extension_name: ""`                      | Validation error: "bridge instance extension name is required"   |
| Missing display_name                 | `display_name: ""`                        | Validation error: "bridge instance display name is required"     |
| Invalid scope                        | `scope: "tenant"`                         | Validation error: unsupported scope                              |
| Workspace scope without workspace_id | `scope: "workspace", workspace_id: ""`    | Validation error: "workspace scope requires workspace id"        |
| Global scope with workspace_id       | `scope: "global", workspace_id: "ws-001"` | Validation error: "global scope cannot include workspace id"     |
| Invalid JSON in provider_config      | `provider_config: "not-json"`             | Validation error: must be valid JSON                             |
| Invalid JSON in delivery_defaults    | `delivery_defaults: "{bad"`               | Validation error: must be valid JSON                             |
| Enabled=true with status=disabled    | `enabled: true, status: "disabled"`       | Validation error: enabled instance cannot report disabled status |
| Enabled=false with status=ready      | `enabled: false, status: "ready"`         | Validation error: disabled instance must report disabled status  |
| Invalid DM policy                    | `dm_policy: "block_all"`                  | Validation error: unsupported dm policy                          |
| Unsupported source                   | `source: "imported"`                      | Validation error: unsupported bridge instance source             |

### Related Test Cases

- TC-FUNC-002 (enable/start lifecycle)
- TC-FUNC-003 (update)
- TC-FUNC-005 (state machine transitions)
- TC-FUNC-006 (provider_config vs delivery_defaults)
