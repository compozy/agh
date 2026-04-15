# Slack Bridge Provider

`extensions/bridges/slack` is the production Slack bridge provider for AGH. It runs as a provider-scoped subprocess on top of `internal/bridgesdk` and multiplexes one or more owned `BridgeInstance` records inside a single Slack runtime.

It implements:

- provider-scoped Host API ownership through `bridges/instances/list`, `bridges/instances/get`, `bridges/instances/report_state`, and `bridges/messages/ingest`
- hardened webhook ingress with method/content-type/body-size/rate-limit/in-flight checks plus Slack signing-secret verification
- Slack Events API messages plus typed bridge `command`, `action`, and `reaction` ingest flows
- outbound `chat.postMessage`, `chat.update`, and `chat.delete` behavior for bridge delivery requests
- restart-safe resume handling through the shared bridge delivery broker

## Build

From the repository root:

```bash
go build -o ./extensions/bridges/slack/bin/slack ./extensions/bridges/slack
```

## Install

Build the binary first, then install the extension directory:

```bash
agh extension install ./extensions/bridges/slack
```

## Provider Config

The bridge instance `provider_config` JSON object currently supports:

```json
{
  "api_base_url": "https://slack.com/api",
  "webhook": {
    "listen_addr": "127.0.0.1:8080",
    "path": "/slack/brg-main"
  },
  "dm": {
    "allow_user_ids": ["U12345"],
    "allow_usernames": ["alice"],
    "paired_user_ids": ["U12345"],
    "paired_usernames": ["alice"]
  },
  "batching": {
    "delay_ms": 0,
    "split_delay_ms": 0,
    "split_threshold": 0
  }
}
```

Notes:

- `bot_token` and `signing_secret` are required through bridge secret bindings.
- `AGH_BRIDGE_SLACK_LISTEN_ADDR` and `AGH_BRIDGE_SLACK_API_BASE_URL` can provide process-level defaults for local development and integration tests.
- Direct-message enforcement uses the bridge instance `dm_policy` plus the provider-config allowlist or paired-user fields.
