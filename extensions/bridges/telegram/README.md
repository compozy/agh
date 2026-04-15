# Telegram Bridge Provider

`extensions/bridges/telegram` is the first production bridge provider for AGH. It runs as a provider-scoped subprocess on top of `internal/bridgesdk` and multiplexes one or more owned `BridgeInstance` records inside a single Telegram runtime.

It implements:

- provider-scoped Host API ownership through `bridges/instances/list`, `bridges/instances/get`, `bridges/instances/report_state`, and `bridges/messages/ingest`
- hardened webhook ingress with method/content-type/body-size/rate-limit/in-flight checks plus Telegram secret-token verification
- direct-chat and group/forum routing identity mapping into bridge v1 inbound envelopes
- outbound `sendMessage`, `editMessageText`, and `deleteMessage` behavior for bridge delivery requests
- restart-safe resume handling through the shared bridge delivery broker

## Build

From the repository root:

```bash
go build -o ./extensions/bridges/telegram/bin/telegram ./extensions/bridges/telegram
```

## Install

Build the binary first, then install the extension directory:

```bash
agh extension install ./extensions/bridges/telegram
```

## Provider Config

The bridge instance `provider_config` JSON object currently supports:

```json
{
  "api_base_url": "https://api.telegram.org",
  "webhook": {
    "listen_addr": "127.0.0.1:8080",
    "path": "/telegram/brg-main"
  },
  "dm": {
    "allow_user_ids": ["12345"],
    "allow_usernames": ["alice"],
    "paired_user_ids": ["12345"],
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

- `bot_token` is required through bridge secret bindings.
- `webhook_secret` is optional; when set, inbound requests must include `X-Telegram-Bot-Api-Secret-Token`.
- `AGH_BRIDGE_TELEGRAM_LISTEN_ADDR` and `AGH_BRIDGE_TELEGRAM_API_BASE_URL` can provide process-level defaults for local development and integration tests.
- Direct-message enforcement uses the bridge instance `dm_policy` plus the provider-config allowlist or paired-user fields.
