# Adding an In-Tree Bridge Provider

This guide covers a provider shipped under `extensions/bridges/<provider>`. A complete provider is
an extension manifest, a subprocess adapter, provider-owned verification, focused tests, operator
documentation, official-skill guidance, and QA tracker impact. There is no central provider list:
the extension catalog and conformance suite discover valid provider directories automatically.

Use the public [bridge-author walkthrough](../../packages/site/content/runtime/core/bridges/adding-a-bridge.mdx)
for the linear build/install/create/verify/send path. This file is the concise in-repo implementation
and review checklist. It is not an external SDK guide; in-tree providers may import
`internal/bridgesdk`, external modules may not.

Reference owners:

| Need                                         | Owner                                    |
| -------------------------------------------- | ---------------------------------------- |
| CI-safe protocol/fake provider               | `sdk/examples/telegram-reference`        |
| Modern lifecycle/reconciler/HTTP composition | `extensions/bridges/slack/provider.go`   |
| Remote webhook registration/control runtime  | `extensions/bridges/telegram/control.go` |
| Conditional ingress modes                    | `extensions/bridges/gchat`               |
| Issue-side no-op progress                    | `extensions/bridges/github`, `linear`    |

## Architecture boundary

```text
platform → authenticated provider ingress → bridges/messages/ingest → workspace route/session
platform ← provider API                ← bridges/deliver         ← ordered delivery broker
                         provider subprocess ↔ AGH over JSON-RPC/stdio
```

AGH owns workspace routing, sessions, persistence, delivery ordering, and extension lifecycle. The
provider owns platform authentication, event normalization, target semantics, API calls, limits,
formatting, and unsupported-operation truth.

| Shared bridgesdk owner                                                              | Provider responsibility                                                           |
| ----------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| `Runtime`: RPC methods, session/cache, default target snapshots, no-op progress ACK | Deliver, check, optional webhook registration, provider health                    |
| `ProviderLifecycle`: Host sync, state reporting, goroutines, shutdown join          | Reconciliation states, resource startup/cleanup, degradation reason               |
| `ManagedConfigReconciler` + `RouteTable`: atomic full-snapshot publication          | Config decode/validation, ownership keys, path conflicts, live probes             |
| `ProviderHTTPServer`: listener/server lifecycle                                     | Raw-body auth, method/content type, size/rate/in-flight bounds, response contract |
| `ProviderHost`: typed instance/state/ingest calls                                   | The point after auth, ACL, dedup, and validation where ingestion is safe          |
| `DeliveryStateStore`: typed provider-local snapshots                                | Create/edit/delete/resume rules and acknowledgement anchors                       |

## 1. Define the contract first

Choose one lowercase provider key and keep it identical across:

- directory name;
- extension `name`;
- bridge `platform`;
- subprocess/runtime extension name;
- config schema `agh.bridge.<provider>`;
- CLI and documentation examples.

Classify each credential before coding. Required slots must be required for every mode. Credentials
needed only by one auth or ingress mode stay optional in the manifest and become explicit conditional
requirements in runtime validation and setup docs.

Decide these provider-owned behaviors before implementation:

- inbound transport, signature/authentication, event allowlist, deduplication, and DM policy;
- route identity (`peer_id`, `group_id`, `thread_id`) and reply/edit context;
- outbound create/edit/delete capabilities, limits, formatting, and retry classification;
- identity/configuration checks and whether webhook registration is implementable;
- progress behavior: render with `ProgressAccumulator` or accept the SDK's no-op acknowledgement;
- target snapshot behavior and workspace/session/agent scope of every datum.

## 2. Create `extension.toml`

Start from this manifest and replace every `acme-chat` value and slot description:

```toml
[extension]
name = "acme-chat"
version = "0.1.0"
description = "Acme Chat bridge provider built on internal/bridgesdk"
min_agh_version = "0.5.0"

[capabilities]
provides = ["bridge.adapter"]

[bridge]
platform = "acme-chat"
display_name = "Acme Chat"

[[bridge.secret_slots]]
name = "bot_token"
description = "Acme Chat bot token"
required = true

[[bridge.secret_slots]]
name = "webhook_secret"
description = "Acme Chat webhook signing secret"
required = true

[bridge.config_schema]
schema = "agh.bridge.acme-chat"
version = "1"

[actions]
requires = [
  "bridges/instances/list",
  "bridges/instances/get",
  "bridges/instances/report_state",
  "bridges/messages/ingest",
]

[subprocess]
command = "./bin/acme-chat"
args = ["serve"]

[subprocess.env]
AGH_BRIDGE_ACME_CHAT_LISTEN_ADDR = "{{env:AGH_BRIDGE_ACME_CHAT_LISTEN_ADDR}}"

[security]
capabilities = ["bridge.read", "bridge.write"]
```

Do not put credential-bearing upstream destinations in `provider_config`. Use fixed official
defaults and explicit operator-owned process variables for trusted sovereign/test overrides. The
daemon rejects instance fields such as `api_base_url`, `oauth_token_url`, `service_url`,
`openid_metadata_url`, and `token_url`.

## 3. Bootstrap the subprocess

This minimal `main.go` compiles and participates in lifecycle conformance. It intentionally returns
an error for delivery until the provider implementation is complete; never ship a false-success
acknowledgement.

```go
package main

import (
    "context"
    "errors"
    "io"

    bridgepkg "github.com/compozy/agh/internal/bridges"
    "github.com/compozy/agh/internal/bridgesdk"
    "github.com/compozy/agh/internal/subprocess"
)

func main() {
    bridgesdk.Main("acme-chat", serve)
}

func serve(stdin io.Reader, stdout io.Writer, _ io.Writer) error {
    lifecycle, err := bridgesdk.NewProviderLifecycle(bridgesdk.ProviderLifecycleConfig{
        ProviderName: "acme-chat",
    })
    if err != nil {
        return err
    }

    runtime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
        ExtensionInfo: subprocess.InitializeExtensionInfo{
            Name: "acme-chat", Version: "0.1.0", SDKName: "bridgesdk",
        },
        Initialize: lifecycle.Initialize,
        Deliver: func(
            context.Context,
            *bridgesdk.Session,
            bridgepkg.DeliveryRequest,
        ) (bridgepkg.DeliveryAck, error) {
            return bridgepkg.DeliveryAck{}, errors.New("acme-chat: delivery not implemented")
        },
        Check: func(
            _ context.Context,
            _ *bridgesdk.Session,
            _ bridgepkg.BridgeCheckRequest,
        ) (bridgepkg.BridgeCheckResponse, error) {
            return bridgepkg.BridgeCheckResponse{
                Checks: []bridgepkg.BridgeCheckRecord{
                    bridgepkg.SkippedCheck(
                        "provider.identity",
                        "Implement the Acme Chat identity probe before enabling this provider.",
                    ),
                },
            }, nil
        },
        HealthCheck: func(context.Context, *bridgesdk.Session) error {
            return lifecycle.Health()
        },
        Shutdown: lifecycle.Shutdown,
    })
    if err != nil {
        return err
    }
    return lifecycle.Serve(context.Background(), runtime, stdin, stdout)
}
```

Build output lives in an ignored directory that does not exist in a fresh checkout:

```bash
mkdir -p ./extensions/bridges/acme-chat/bin
go build \
  -o ./extensions/bridges/acme-chat/bin/acme-chat \
  ./extensions/bridges/acme-chat
```

The bootstrap is a compile/lifecycle checkpoint. Delivery fails truthfully and identity is an
explicit `skipped` check until provider behavior is implemented. An empty check response is invalid.

Split production code by responsibility before it grows: config resolution, webhook authentication,
inbound mapping, API client, delivery, progress, control checks, and process entry. Production files
must stay below 500 lines.

## 4. Wire shared lifecycle owners

Use the shared owners instead of copying another provider's scaffolding:

- `ProviderLifecycle` owns initialize, Host API synchronization, state reporting, goroutines, health,
  and cooperative shutdown.
- `ManagedConfigReconciler` resolves a full managed-instance snapshot, publishes routes atomically,
  runs provider-specific probes, and removes retired state.
- `RouteTable` owns instance and webhook-path lookup.
- `ProviderHTTPServer` owns one listener and lifecycle-bound serving/shutdown.
- `DeliveryStateStore` owns provider-specific in-memory delivery snapshots; the daemon broker owns
  durable checkpoints.
- `ProviderHost` owns typed instance list/get/report-state calls.
- `ProgressAccumulator` and `ProgressDispatcher` own editable progress bubbles and throttled flushes.
- `RunProviderCommand`/`Main` own the `serve` command surface.

The adapter still owns platform truth. Decode and validate `provider_config`, bind exact manifest
slots from `Session.Cache()`, authenticate every inbound request before ingestion, and classify API
failures with bridgesdk's typed auth/rate-limit/transient/permanent errors.

## 5. Implement the runtime contract

At minimum:

1. Reconcile every managed instance and report `ready`, `degraded`, `auth_required`, or `error`.
2. Start a bounded webhook listener or polling loop under `ProviderLifecycle.Go`.
3. Authenticate, size-limit, rate-limit, deduplicate, map, and validate inbound events.
4. Ingest through `ProviderLifecycle.Host()` only after ACL/DM policy accepts the event.
5. Implement `bridges/deliver` with stable route targets and a validated acknowledgement.
6. Return the last materialized remote ID for multi-part terminal delivery.
7. Preserve explicit edit references over local state; never let progress IDs become text anchors.
8. Implement provider-owned `Check`; add `RegisterWebhook` only when the provider supports remote
   registration.
9. Return truthful target snapshots or keep the SDK's managed-instance fallback.
10. Reap listeners, timers, batchers, and goroutines on every shutdown and initialization failure.

The default SDK behavior acknowledges progress and a progress-only empty final without calling the
text handler. Add progress rendering only when the platform supports it truthfully.

## 6. Run the public lifecycle once

```bash
agh extension install ./extensions/bridges/acme-chat
agh extension status acme-chat -o json

agh bridge create \
  --scope workspace \
  --workspace-id "$WORKSPACE_ID" \
  --platform acme-chat \
  --extension acme-chat \
  --display-name "Acme Chat development" \
  --enabled=false \
  --provider-config '{"webhook":{"listen_addr":"127.0.0.1:18090","path":"/acme/dev"}}' \
  -o json
```

Copy the ID into `BRIDGE_ID`, bind every required slot, then prove the exact sequence:

```bash
agh bridge secret-bindings list "$BRIDGE_ID" -o json
agh bridge verify "$BRIDGE_ID" --json
agh bridge enable "$BRIDGE_ID"
# Send one authenticated fake-provider event.
agh bridge routes "$BRIDGE_ID" -o json
agh bridge targets "$BRIDGE_ID" -o json
agh bridge send-test "$BRIDGE_ID" --peer-id "$ACME_CHAT_PEER_ID" \
  --message "AGH bridge provider smoke test" --json
```

Completion requires a `ready` provider, a route in `$WORKSPACE_ID`, and a fake-platform delivery ID
plus remote message ID. Do not treat build or initialize alone as a functional provider.

## 7. Add focused tests

Before adding a test, name its invariant, owning layer, and canonical suite. Reuse the provider's
`provider_test.go`, `provider_delivery_test.go`, `control_test.go`, and integration owner rather than
forking the same assertion across layers.

Cover at least:

- manifest/config/slot validation and each conditional auth mode;
- valid and invalid webhook authentication over the exact body;
- accepted/ignored event mapping, DM policy, deduplication, and workspace propagation;
- create/edit/delete/resume behavior, length limits, retry classification, and acknowledgements;
- provider-owned checks, disabled reachability, lifecycle reconciliation, and cleanup;
- progress rendering or explicit no-side-effect acknowledgement.

Run focused gates from the repository root:

```bash
go test -race ./extensions/bridges/acme-chat/...
go test -race -tags=integration ./internal/extension \
  -run '^TestAutoDiscoveredProviderRuntimeConformance$'
go test -race ./internal/extension \
  -run '^TestBridgeProviderDocsConformance$'
```

## 8. Co-ship operator and agent surfaces

Update in the same change:

- `packages/site/content/runtime/core/bridges/index.mdx`: exact display name and slot row;
- `packages/site/content/runtime/core/bridges/setup-acme-chat.mdx`: behavior-first CLI setup,
  conditional slots, provider-side callback, verify/enable/send-test, and troubleshooting;
- `packages/site/content/runtime/core/bridges/meta.json`: navigation entry;
- provider `README.md`: build, config, transport, limits, and unsupported operations;
- `skills/agh/`: public provider behavior and agent-manageable CLI/HTTP/UDS path;
- `docs/qa/state.csv`: add new user-visible behavior as `untested`.

The setup page is a working how-to, not a config dump. It must show where each provider credential
comes from, the provider-console steps, public-to-local endpoint mapping, disabled and enabled verify
checkpoints, a real inbound route plus `send-test`, configuration defaults, known limits, security,
and troubleshooting by observable symptom.

If the change modifies shared bridge contracts, CLI/API routes, native tools, or generated schemas,
co-ship OpenAPI, TypeScript SDK/Web contracts, CLI reference, mocks, and official skill updates. A
provider-only implementation must not invent a provider registry or `config.toml` key.

## 9. Grep verification recipe

Run this before review and inspect every match:

```bash
PROVIDER=acme-chat
DISPLAY='Acme Chat'

rg -n "$PROVIDER|$DISPLAY" \
  "extensions/bridges/$PROVIDER" \
  packages/site/content/runtime/core/bridges \
  skills/agh docs/qa/state.csv

rg -n 'bridge\.adapter|bridge\.secret_slots|bridge\.config_schema|bridges/deliver' \
  "extensions/bridges/$PROVIDER"

rg -n 'api_base_url|oauth_token_url|service_url|openid_metadata_url|token_url' \
  "extensions/bridges/$PROVIDER"

rg -n 'TODO|FIXME|not implemented|chat\.postMessage|chat\.update|chat\.delete' \
  "extensions/bridges/$PROVIDER" \
  "packages/site/content/runtime/core/bridges/setup-$PROVIDER.mdx"
```

The first two searches prove discoverability and cross-surface coverage. The third requires every
match to be an operator-owned environment seam or a rejection test. The final search catches
unfinished code and copied provider terminology; an intentional unsupported-operation error must be
reviewed rather than deleted blindly.

## Review exit criteria

| Requirement                                            | Evidence owner                                |
| ------------------------------------------------------ | --------------------------------------------- |
| Manifest identity, slots, schema, discovery            | Auto-discovered provider conformance          |
| Initialize, methods, health, cooperative shutdown      | Runtime conformance                           |
| Auth, mapping, DM policy, dedup, workspace propagation | Provider suite                                |
| Delivery operations, limits, retry class, ACK          | Provider delivery suite                       |
| Check/register behavior without Host API/listener      | Control suite                                 |
| Failure/cancellation/removal cleanup                   | Race-enabled lifecycle and failure-path cases |
| Setup page, slots, navigation                          | Docs conformance and focused site validation  |
| CLI/HTTP/UDS and official skill                        | Cross-surface impact audit                    |

The Web setup profile remains a bundled-provider presentation surface today. Audit
`web/src/systems/bridges/lib/bridge-setup.ts`; dynamic manifest discovery does not by itself create a
full guided Web flow for a new provider.

## Author troubleshooting

| Symptom                                       | Check                                                                                                     |
| --------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| Provider is absent from the catalog           | Manifest identity, install/enable state, `bridge.adapter`, subprocess path.                               |
| Build cannot write `bin/<provider>`           | Create the ignored `bin/` directory before `go build -o`.                                                 |
| Initialize rejects methods/grants             | Manifest actions, runtime registration, extension identity, handshake purpose.                            |
| Verify response is invalid                    | Return at least one unique valid check; empty checks and missing remediation on fail/skipped are invalid. |
| Verify sees nil Host API or starts a listener | Control handlers must use `Session.Cache()` only and must not run service initialization.                 |
| Workspace ingest fails or crosses ownership   | Copy scope/workspace ID from the managed instance and validate the typed envelope.                        |
| `send-test` cannot resolve                    | Target snapshot/fallback, canonical route, real inbound route, and `targets`/`resolve` output.            |
| Shutdown hangs                                | Goroutine/listener/timer/batcher/poller ownership outside `ProviderLifecycle`.                            |
