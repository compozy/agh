## TC-INT-007: HTTP API Bridge CRUD Operations

**Priority:** P1
**Type:** Integration
**Systems:** api/httpapi (Gin router), api/core handlers, api/contract types, bridges.Registry, bridges.BridgeInstance, bridges.RoutingPolicy, store/globaldb
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-15

---

### Objective
Validate the complete HTTP API surface for bridge instance management: POST create, GET by ID, GET list (with scope/platform filters), PATCH update. Confirms that request bodies round-trip through contract types correctly, responses match the API contract shape, and `provider_config` / `delivery_defaults` JSON blobs survive serialization.

### Preconditions
- [ ] HTTP test server running via `httptest.NewServer` with Gin router configured
- [ ] globaldb initialized via `t.TempDir()`
- [ ] At least 1 bridge-capable extension is registered with the extension manager (for platform validation)
- [ ] No bridge instances exist in globaldb before test starts

### Test Steps
1. **POST /api/bridges - Create a global bridge instance**
   - Input: `POST /api/bridges` with JSON body: `{"scope": "global", "platform": "telegram", "extension_name": "telegram-adapter", "display_name": "Test TG Bridge", "enabled": true, "status": "starting", "routing_policy": {"include_peer": true, "include_thread": false, "include_group": false}, "provider_config": {"bot_token_ref": "env:TG_TOKEN"}, "delivery_defaults": {"parse_mode": "markdown"}}`
   - **Expected:** HTTP 201; response body contains `id` (non-empty UUID/slug), `scope=global`, `platform=telegram`, `extension_name=telegram-adapter`, `display_name="Test TG Bridge"`, `enabled=true`, `status=starting`, `routing_policy.include_peer=true`, `provider_config` round-tripped, `delivery_defaults` round-tripped, `created_at` and `updated_at` are recent ISO8601 timestamps

2. **POST /api/bridges - Create a workspace-scoped bridge instance**
   - Input: `POST /api/bridges` with JSON body: `{"scope": "workspace", "workspace_id": "ws-prod", "platform": "slack", "extension_name": "slack-adapter", "display_name": "Prod Slack", "enabled": false, "status": "disabled", "routing_policy": {"include_peer": true, "include_thread": true, "include_group": false}}`
   - **Expected:** HTTP 201; `scope=workspace`, `workspace_id=ws-prod`, `enabled=false`, `status=disabled`

3. **GET /api/bridges/:id - Retrieve bridge by ID**
   - Input: `GET /api/bridges/<id_from_step1>`
   - **Expected:** HTTP 200; response body matches step 1 creation response exactly; `provider_config` and `delivery_defaults` JSON objects are identical

4. **GET /api/bridges - List all bridges (no filters)**
   - Input: `GET /api/bridges`
   - **Expected:** HTTP 200; response is a JSON array with 2 items (global telegram, workspace slack)

5. **GET /api/bridges?scope=global - List with scope filter**
   - Input: `GET /api/bridges?scope=global`
   - **Expected:** HTTP 200; response array has 1 item (telegram bridge only)

6. **GET /api/bridges?platform=slack - List with platform filter**
   - Input: `GET /api/bridges?platform=slack`
   - **Expected:** HTTP 200; response array has 1 item (slack bridge only)

7. **PATCH /api/bridges/:id - Update display name and routing policy**
   - Input: `PATCH /api/bridges/<id_from_step1>` with JSON body: `{"display_name": "Renamed TG Bridge", "routing_policy": {"include_peer": true, "include_thread": true, "include_group": false}}`
   - **Expected:** HTTP 200; response body shows `display_name="Renamed TG Bridge"`, `routing_policy.include_thread=true`, `updated_at` is newer than `created_at`

8. **PATCH /api/bridges/:id - Update delivery_defaults**
   - Input: `PATCH /api/bridges/<id_from_step1>` with JSON body: `{"delivery_defaults": {"parse_mode": "html", "disable_preview": true}}`
   - **Expected:** HTTP 200; `delivery_defaults` round-trips the new JSON object; old `provider_config` is unchanged

9. **PATCH /api/bridges/:id - Clear delivery_defaults with null**
   - Input: `PATCH /api/bridges/<id_from_step1>` with JSON body: `{"delivery_defaults": null}`
   - **Expected:** HTTP 200; `delivery_defaults` is null or absent in response

10. **GET /api/bridges/:id - Verify final state**
    - Input: `GET /api/bridges/<id_from_step1>`
    - **Expected:** Reflects all accumulated updates: renamed display name, updated routing policy, cleared delivery_defaults

### Data Validation
| Field | Source Value | Transformed Value | Status |
|-------|------------|-------------------|--------|
| Request scope=global | JSON string | BridgeInstance.Scope = ScopeGlobal | |
| Request routing_policy.include_peer=true | JSON boolean | BridgeInstance.RoutingPolicy.IncludePeer = true | |
| Request provider_config | JSON object | json.RawMessage round-trip | |
| Request delivery_defaults | JSON object | json.RawMessage round-trip | |
| Request delivery_defaults=null | JSON null | BridgeInstance.DeliveryDefaults = nil | |
| Response created_at | Server timestamp | ISO8601 string, within 5s of now | |
| Response updated_at after PATCH | Server timestamp | Strictly newer than created_at | |

### Error Scenarios
- [ ] POST with missing required field (platform=""): HTTP 400 with validation error
- [ ] POST with scope=workspace but no workspace_id: HTTP 400 (ValidateScopeWorkspaceID fails)
- [ ] POST with unsupported scope value: HTTP 400
- [ ] GET with non-existent ID: HTTP 404
- [ ] PATCH with invalid routing_policy (include_thread without include_peer or include_group): HTTP 400
- [ ] PATCH with invalid JSON in delivery_defaults (non-object): HTTP 400
- [ ] POST with invalid provider_config JSON: HTTP 400
- [ ] PATCH on a managed (source=package) instance: HTTP 403 or 400 (ErrBridgeInstanceReadOnly)

### Related Test Cases
- TC-INT-008 (UDS API should return same data as HTTP API)
- TC-INT-009 (CLI commands call through to the same API)
