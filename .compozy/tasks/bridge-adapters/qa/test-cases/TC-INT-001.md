## TC-INT-001: Provider Runtime Launch with Multiple Instances

**Priority:** P0
**Type:** Integration
**Systems:** bridgesdk.Runtime, extension.Manager, subprocess (JSON-RPC over stdio), bridgesdk.InstanceCache, store/globaldb
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-15

---

### Objective
Validate that a bridge-capable extension runtime launches via the 6-phase extension pipeline (Discover, Parse, Validate, Register, Initialize, Activate), receives an `InitializeBridgeRuntime` payload containing multiple managed instances with resolved bound secrets, populates the `InstanceCache`, and exposes all instances through the Host API `bridges/instances/list` call.

### Preconditions
- [ ] Extension manifest declares `provides: ["bridge"]` with a valid `bridge_provider` block
- [ ] globaldb contains 3 enabled `BridgeInstance` rows with `status=starting`, `source=package`, each bound to the same `extension_name` and `platform`
- [ ] Each instance has at least 1 `BridgeSecretBinding` row with a resolvable `vault_ref`
- [ ] No other extension process is running for the same provider
- [ ] SQLite test database initialized via `t.TempDir()`

### Test Steps
1. **Seed globaldb with 3 bridge instances and their secret bindings**
   - Input: BridgeInstance rows: `brg-inst-1` (scope=global, platform=telegram, routing_policy={peer:true}), `brg-inst-2` (scope=workspace, workspace_id=ws-1, platform=telegram, routing_policy={peer:true, thread:true}), `brg-inst-3` (scope=global, platform=telegram, routing_policy={peer:true, group:true})
   - **Expected:** All 3 rows inserted without error; each has 1+ BridgeSecretBinding with `kind=env`

2. **Start the extension manager with the bridge runtime resolver**
   - Input: Extension manifest path, bridge runtime resolver that returns the 3 seeded instances
   - **Expected:** Manager completes Discover->Parse->Validate->Register phases without error

3. **Verify the `initialize` JSON-RPC request sent to the subprocess**
   - Input: Capture the raw JSON-RPC params from the subprocess stdin
   - **Expected:** `request.runtime.bridge` is non-nil; `request.runtime.bridge.provider` matches extension name; `request.runtime.bridge.platform` equals `telegram`; `request.runtime.bridge.managed_instances` has length 3

4. **Verify bound secrets are resolved in each managed instance**
   - Input: Inspect `managed_instances[*].bound_secrets` in the initialize request
   - **Expected:** Each managed instance carries at least 1 `BoundSecret` entry with non-empty `binding_name` and `value` fields; values match the vault-resolved plaintext

5. **Verify the InstanceCache is populated after handshake**
   - Input: Call `runtime.Session().Cache().List()`
   - **Expected:** Returns 3 `InitializeBridgeManagedInstance` entries; IDs match `brg-inst-1`, `brg-inst-2`, `brg-inst-3`

6. **Verify Host API `bridges/instances/list` returns all 3 instances**
   - Input: Call `session.HostAPI().ListBridgeInstances(ctx)`
   - **Expected:** Returns 3 `BridgeInstance` entries with correct `id`, `platform`, `scope`, `workspace_id`, `routing_policy`, and `status`

7. **Verify the initialize response confirms bridge capability**
   - Input: Inspect `session.InitializeResponse()`
   - **Expected:** `accepted_capabilities.provides` contains `"bridge"`; `implemented_methods` contains `"bridges/deliver"`

### Data Validation
| Field | Source Value | Transformed Value | Status |
|-------|------------|-------------------|--------|
| runtime.bridge.provider | extension manifest `name` | InitializeBridgeRuntime.Provider | |
| runtime.bridge.platform | extension manifest `bridge_provider.platform` | InitializeBridgeRuntime.Platform | |
| managed_instances[0].instance.id | globaldb `brg-inst-1` | InitializeBridgeManagedInstance.Instance.ID | |
| managed_instances[0].bound_secrets[0].value | vault plaintext | InitializeBridgeBoundSecret.Value | |
| managed_instances[1].instance.scope | globaldb `workspace` | BridgeInstance.Scope = ScopeWorkspace | |
| managed_instances[1].instance.workspace_id | globaldb `ws-1` | BridgeInstance.WorkspaceID = "ws-1" | |
| managed_instances[2].instance.routing_policy | globaldb `{peer:true, group:true}` | BridgeInstance.RoutingPolicy | |

### Error Scenarios
- [ ] Extension manifest missing `bridge_provider` block: manager returns `ErrBridgeRuntimeResolverRequired`
- [ ] All 3 instances have `enabled=false` and `status=disabled`: manager defers launch with `ErrBridgeRuntimeDeferred`
- [ ] Vault reference unresolvable for one binding: initialize fails with descriptive error, no partial cache population
- [ ] Subprocess exits during initialize handshake: manager detects process exit and does not populate session
- [ ] Duplicate initialize call after successful handshake: returns RPC error code "already initialized"

### Related Test Cases
- TC-INT-003 (multi-instance routing isolation depends on successful multi-instance launch)
- TC-INT-004 (delivery requires a launched provider with cached instances)
- TC-INT-010 (managed instance sync uses the same globaldb rows)
