## TC-FUNC-019: Provider Manifest Metadata

**Priority:** P2
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective
Verify that provider manifests for all 8 bridge providers correctly declare `bridge.platform`, `bridge.display_name`, and the required secret slots, matching the provider specifications from the techspec.

### Preconditions
- [ ] All 8 provider extensions are compiled and registered
- [ ] Provider manifests are accessible through the extension manager or provider enumeration API
- [ ] The techspec secret slot table is the reference source

### Test Steps
1. **Verify Telegram provider manifest**
   - **Expected:**
     - `platform` = `"telegram"`
     - `display_name` is present and non-empty (e.g., `"Telegram"`)
     - Secret slots include:
       - `bot_token` (required)
       - `webhook_secret` (optional)

2. **Verify Slack provider manifest**
   - **Expected:**
     - `platform` = `"slack"`
     - `display_name` is present (e.g., `"Slack"`)
     - Secret slots include:
       - `bot_token` (required)
       - `signing_secret` (required)

3. **Verify Discord provider manifest**
   - **Expected:**
     - `platform` = `"discord"`
     - `display_name` is present (e.g., `"Discord"`)
     - Secret slots include:
       - `bot_token` (required)
       - `public_key` (required)

4. **Verify WhatsApp provider manifest**
   - **Expected:**
     - `platform` = `"whatsapp"`
     - `display_name` is present (e.g., `"WhatsApp"`)
     - Secret slots include:
       - `access_token` (required)
       - `app_secret` (required)
       - `verify_token` (required)

5. **Verify Teams provider manifest**
   - **Expected:**
     - `platform` = `"teams"`
     - `display_name` is present (e.g., `"Microsoft Teams"`)
     - Secret slots include:
       - `app_id` (required)
       - `app_password` (required)
       - `app_tenant_id` (optional)

6. **Verify Google Chat provider manifest**
   - **Expected:**
     - `platform` = `"gchat"`
     - `display_name` is present (e.g., `"Google Chat"`)
     - Secret slots include:
       - `credentials_json` (required)
       - `project_number` (required)

7. **Verify GitHub provider manifest**
   - **Expected:**
     - `platform` = `"github"`
     - `display_name` is present (e.g., `"GitHub"`)
     - Secret slots include:
       - `webhook_secret` (required)
       - `token` (required for PAT mode)
       - `app_id` (required for App mode)
       - `private_key` (required for App mode)

8. **Verify Linear provider manifest**
   - **Expected:**
     - `platform` = `"linear"`
     - `display_name` is present (e.g., `"Linear"`)
     - Secret slots include:
       - `webhook_secret` (required)
       - `api_key` (required for single-tenant mode)
       - `client_id` (required for OAuth mode)
       - `client_secret` (required for OAuth mode)

9. **Verify all manifests have valid BridgeSecretSlot structure**
   - Input: For each provider, iterate over secret slots
   - **Expected:** Each slot has a non-empty `name`, optional `description`, and `required` boolean. `BridgeSecretSlot.Validate()` passes for every slot.

10. **Verify optional config_schema hints**
    - Input: Check each provider for `BridgeProviderConfigSchema`
    - **Expected:** If present, `config_schema.schema` or `config_schema.version` is non-empty. `BridgeProviderConfigSchema.Validate()` passes.

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Secret slot with empty name | `BridgeSecretSlot{Name: ""}` | Validation error: "bridge secret slot name is required" |
| Duplicate secret slot names | Two slots with `name: "bot_token"` | Should be rejected or deduplicated |
| Provider with no secret slots | Hypothetical provider with zero slots | Valid but unusual; manifest still requires platform and display_name |
| Config schema with neither schema nor version | `BridgeProviderConfigSchema{Schema: "", Version: ""}` (non-zero but empty) | Validation error or normalized to zero value |

### Related Test Cases
- TC-FUNC-001 (creation uses platform from manifest)
- TC-FUNC-006 (provider_config relates to manifest schema)
