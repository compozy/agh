## TC-SEC-010: Provider Config Injection Prevention

**Priority:** P2
**Type:** Security
**Risk Level:** Medium
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-15

---

### Objective
Verify that provider configuration (`provider_config`) and delivery defaults (`delivery_defaults`) are strictly separated, preventing config injection attacks where an attacker manipulates one field space to influence the other, and that invalid or unexpected configuration fields are rejected at validation.

### Preconditions
- [ ] Bridge adapter runtime is running with at least one provider type available (e.g., Slack)
- [ ] Host API or configuration endpoint is accessible for creating/updating instances
- [ ] Knowledge of the valid `provider_config` fields (e.g., `bot_token`, `signing_secret`, `app_id`) and `delivery_defaults` fields (e.g., `channel_id`, `thread_ts`, `response_type`)
- [ ] Ability to inspect the resulting instance configuration after creation

### Test Steps

1. **Normal instance creation — fields correctly separated**
   - Input: Create an instance with:
     ```json
     {
       "provider_config": {"bot_token": "xoxb-test", "signing_secret": "secret123"},
       "delivery_defaults": {"channel_id": "C001", "response_type": "in_channel"}
     }
     ```
   - **Expected:** Instance created successfully. `provider_config` contains only `bot_token` and `signing_secret`. `delivery_defaults` contains only `channel_id` and `response_type`. No cross-contamination.

2. **Inject provider_config field into delivery_defaults**
   - Input: Create an instance with:
     ```json
     {
       "provider_config": {"bot_token": "xoxb-test"},
       "delivery_defaults": {"bot_token": "xoxb-injected", "channel_id": "C001"}
     }
     ```
   - **Expected:** Either (a) `bot_token` in `delivery_defaults` is rejected at validation (400 Bad Request), or (b) `bot_token` in `delivery_defaults` is silently ignored and does not override or supplement `provider_config.bot_token`. The instance's operational `bot_token` remains `xoxb-test`.

3. **Inject delivery_defaults field into provider_config**
   - Input: Create an instance with:
     ```json
     {
       "provider_config": {"bot_token": "xoxb-test", "channel_id": "C-injected"},
       "delivery_defaults": {}
     }
     ```
   - **Expected:** Either (a) `channel_id` in `provider_config` is rejected at validation, or (b) `channel_id` in `provider_config` is silently ignored and does not affect delivery behavior. Messages are not sent to `C-injected` unless explicitly set in `delivery_defaults`.

4. **Inject secret into delivery_defaults**
   - Input: Create an instance with:
     ```json
     {
       "provider_config": {"bot_token": "xoxb-real"},
       "delivery_defaults": {"signing_secret": "injected-secret"}
     }
     ```
   - **Expected:** `signing_secret` in `delivery_defaults` is rejected or ignored. Signature verification uses only the value from `provider_config`. The injected value cannot influence security-critical behavior.

5. **Unknown/extra fields in provider_config**
   - Input: Create an instance with:
     ```json
     {
       "provider_config": {"bot_token": "xoxb-test", "malicious_field": "exploit_value"},
       "delivery_defaults": {}
     }
     ```
   - **Expected:** Either (a) unknown field `malicious_field` is rejected at validation (strict schema), or (b) unknown field is silently dropped and never stored or processed. Strict validation preferred.

6. **Unknown/extra fields in delivery_defaults**
   - Input: Create an instance with:
     ```json
     {
       "provider_config": {"bot_token": "xoxb-test"},
       "delivery_defaults": {"__proto__": {"admin": true}, "constructor": "exploit"}
     }
     ```
   - **Expected:** Prototype pollution-style payloads are rejected or safely ignored. No unexpected behavior in the runtime. Fields do not alter object prototypes or internal state.

7. **Nested object injection in config fields**
   - Input: Create an instance with:
     ```json
     {
       "provider_config": {"bot_token": {"nested": "object_instead_of_string"}},
       "delivery_defaults": {}
     }
     ```
   - **Expected:** Validation rejects `bot_token` because it expects a string, not an object. 400 Bad Request with a clear validation error.

8. **Config update does not merge across boundaries**
   - Input: (a) Create an instance with valid `provider_config` and `delivery_defaults`. (b) Update the instance with only `delivery_defaults` changes.
   - **Expected:** Update modifies only `delivery_defaults`. `provider_config` remains unchanged. No partial merge or field leakage between the two configuration spaces during updates.

9. **Config fields with special characters**
   - Input: Create an instance with:
     ```json
     {
       "provider_config": {"bot_token": "xoxb-test"},
       "delivery_defaults": {"channel_id": "C001\"; DROP TABLE instances;--"}
     }
     ```
   - **Expected:** SQL injection payload in `channel_id` is treated as a literal string. No SQL injection. Value is either rejected at validation or safely stored and used as-is (parameterized queries).

10. **Empty provider_config with required fields**
    - Input: Create an instance with:
      ```json
      {
        "provider_config": {},
        "delivery_defaults": {"channel_id": "C001"}
      }
      ```
    - **Expected:** Validation rejects the request because required `provider_config` fields (e.g., `bot_token`) are missing. 400 Bad Request with clear indication of which required fields are absent.

11. **Config retrieval does not expose cross-boundary fields**
    - Input: After creating a valid instance, call `instances/get` to retrieve its configuration.
    - **Expected:** Response clearly separates `provider_config` (with secrets redacted per TC-SEC-007) and `delivery_defaults`. No field migration between the two objects in the response.

12. **Null and zero-value injection**
    - Input: Create an instance with:
      ```json
      {
        "provider_config": {"bot_token": null, "signing_secret": ""},
        "delivery_defaults": {"channel_id": null}
      }
      ```
    - **Expected:** Null values for required fields are rejected at validation. Empty strings for secret fields are either rejected or treated as "not provided." No nil pointer dereference or unexpected behavior downstream.

### Attack Vectors
- [ ] Config injection: delivery_defaults fields overriding provider_config security settings
- [ ] Reverse injection: provider_config fields influencing delivery behavior
- [ ] Prototype pollution via `__proto__`, `constructor`, or `__defineGetter__` keys
- [ ] Type confusion: sending objects/arrays where strings are expected
- [ ] SQL injection via config field values
- [ ] Field migration during config updates merging across boundaries
- [ ] Null/empty value injection bypassing required field validation
- [ ] Unknown field persistence creating shadow configuration state

### Related Test Cases
- TC-SEC-007 (Secret isolation — secrets in provider_config must not leak to delivery_defaults or API responses)
- TC-SEC-006 (Instance ownership — config access is scoped to owning extension)
- TC-SEC-001 (Signature verification — uses provider_config secrets, must not be influenced by delivery_defaults)
