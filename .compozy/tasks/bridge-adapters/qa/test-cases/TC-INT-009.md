## TC-INT-009: CLI Bridge Commands

**Priority:** P0
**Type:** Integration
**Systems:** cli/bridge.go (Cobra commands), cli/client.go (UDS client), api/udsapi, api/contract types, bridges.BridgeInstance, bridges.BridgeRoute, bridges.RoutingPolicy
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-15

---

### Objective
Validate the CLI bridge subcommands end-to-end: `bridge list` (human table + JSON output), `bridge get` (JSON output), `bridge create`, `bridge update`, `bridge enable`, `bridge disable`, `bridge restart`, `bridge routes`, and `bridge test-delivery`. Confirms output formatting, correct flag parsing, scope/platform/status columns in table view, and JSON shape matching the contract types.

### Preconditions
- [ ] Daemon is running with a UDS socket at a known path
- [ ] At least 1 bridge instance exists: `brg-cli-1` (scope=global, platform=telegram, extension_name=telegram-adapter, display_name="CLI Test Bridge", status=ready, enabled=true, routing_policy={IncludePeer:true})
- [ ] A route exists: `brg-cli-1` + `peer_id=peer-1` -> `session_id=sess-1`, `agent_name=claude`
- [ ] CLI binary is available or commands are executed via `cobra.Command.Execute()` in tests

### Test Steps
1. **`agh bridge list` (human output)**
   - Input: Execute `bridge list` command without `--output json`
   - **Expected:** Output is a formatted table with columns: ID, Name, Platform, Extension, Scope, Workspace, Status, Routing, Updated. Row for `brg-cli-1` shows: platform=telegram, status=ready, routing="peer", workspace="-" (dash for empty)

2. **`agh bridge list --output json` (JSON output)**
   - Input: Execute `bridge list --output json`
   - **Expected:** Output is valid JSON array. Each element has keys: `id`, `display_name`, `platform`, `extension_name`, `scope`, `workspace_id`, `status`, `routing_policy`, `enabled`, `created_at`, `updated_at`. Values for `brg-cli-1` match the seeded data

3. **`agh bridge get <id> --output json`**
   - Input: Execute `bridge get brg-cli-1 --output json`
   - **Expected:** Output is valid JSON object with all BridgeRecord fields. `routing_policy.include_peer=true`, `routing_policy.include_thread=false`, `routing_policy.include_group=false`

4. **`agh bridge create`**
   - Input: Execute `bridge create --scope global --platform slack --extension slack-adapter --display-name "New Slack Bridge" --include-peer --delivery-defaults '{"thread_broadcast": true}'`
   - **Expected:** Output shows the created bridge with generated ID, platform=slack, routing includes "peer", delivery_defaults contains the provided JSON

5. **`agh bridge create` with workspace scope**
   - Input: Execute `bridge create --scope workspace --workspace-id ws-dev --platform telegram --extension telegram-adapter --display-name "WS Bridge"`
   - **Expected:** Output shows scope=workspace, workspace_id=ws-dev

6. **`agh bridge create` missing required flag**
   - Input: Execute `bridge create --scope global --display-name "Missing Platform"`
   - **Expected:** Error output indicating `--platform` is required

7. **`agh bridge create` workspace scope without workspace-id**
   - Input: Execute `bridge create --scope workspace --platform telegram --extension telegram-adapter --display-name "No WS ID"`
   - **Expected:** Error: "cli: --workspace-id is required when --scope=workspace"

8. **`agh bridge update <id>`**
   - Input: Execute `bridge update brg-cli-1 --display-name "Updated CLI Bridge" --include-thread`
   - **Expected:** Output shows updated `display_name="Updated CLI Bridge"` and `routing_policy` now includes thread

9. **`agh bridge update` with no flags**
   - Input: Execute `bridge update brg-cli-1`
   - **Expected:** Error: "cli: at least one update flag is required"

10. **`agh bridge enable <id>`**
    - Input: Execute `bridge enable brg-cli-1`
    - **Expected:** Output shows `enabled=true`, `status=starting` (or `ready` depending on current state)

11. **`agh bridge disable <id>`**
    - Input: Execute `bridge disable brg-cli-1`
    - **Expected:** Output shows `enabled=false`, `status=disabled`

12. **`agh bridge routes <id>`**
    - Input: Execute `bridge routes brg-cli-1`
    - **Expected:** Human output: table with columns Hash, Scope, Workspace, Peer, Thread, Group, Session, Agent, Last Active. Shows 1 row with peer=peer-1, session=sess-1, agent=claude

13. **`agh bridge test-delivery <id>`**
    - Input: Execute `bridge test-delivery brg-cli-1 --peer-id peer-1 --mode direct-send --message "test delivery"`
    - **Expected:** Output shows resolved delivery target with bridge_instance_id=brg-cli-1, peer_id=peer-1, mode=direct-send

14. **`agh bridge update <id> --delivery-defaults null`**
    - Input: Execute `bridge update brg-cli-1 --delivery-defaults null`
    - **Expected:** Output shows `delivery_defaults` cleared (null or absent)

15. **`agh bridge update <id> --delivery-defaults 'invalid'`**
    - Input: Execute `bridge update brg-cli-1 --delivery-defaults 'not-json'`
    - **Expected:** Error: "cli: delivery defaults must be valid JSON"

### Data Validation
| Field | Source Value | Transformed Value | Status |
|-------|------------|-------------------|--------|
| Human table "Routing" column | RoutingPolicy{IncludePeer:true} | `"peer"` | |
| Human table "Routing" column | RoutingPolicy{IncludePeer:true, IncludeThread:true} | `"peer, thread"` | |
| Human table "Workspace" column | empty string | `"-"` (dash) | |
| JSON routing_policy | RoutingPolicy struct | `{"include_peer": true, "include_thread": false, "include_group": false}` | |
| JSON updated_at | time.Time | ISO8601/RFC3339 string | |
| Human table "Updated" column | time.Time | Relative age string (e.g., "2m ago") | |
| --delivery-defaults flag | Raw JSON string | json.RawMessage round-trip | |
| --delivery-defaults null | JSON null literal | BridgeInstance.DeliveryDefaults = nil | |

### Error Scenarios
- [ ] `bridge get` with non-existent ID: error message from daemon
- [ ] `bridge create` with invalid --delivery-defaults (array instead of object): "cli: delivery defaults must be a JSON object or null"
- [ ] `bridge update` with --display-name "" (empty): "cli: --display-name cannot be empty"
- [ ] `bridge test-delivery` with invalid --mode: validation error from DeliveryMode.Validate()
- [ ] `bridge create` with --scope=invalid: validation error from Scope.Validate()
- [ ] Daemon not running (UDS socket missing): connection error from client

### Related Test Cases
- TC-INT-007 (HTTP API provides the same CRUD operations)
- TC-INT-008 (UDS API is the transport layer the CLI uses)
