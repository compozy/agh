# WhatsApp Bridge Provider

`extensions/bridges/whatsapp` is the production WhatsApp Cloud API bridge provider for AGH. It runs as a provider-scoped subprocess on top of `internal/bridgesdk` and multiplexes one or more owned `BridgeInstance` records inside a single WhatsApp runtime.

It implements:

- provider-scoped Host API ownership through `bridges/instances/list`, `bridges/instances/get`, `bridges/instances/report_state`, and `bridges/messages/ingest`
- hardened webhook ingress with verify-challenge GET handling plus signed POST validation through `X-Hub-Signature-256`
- direct-message style inbound mapping for WhatsApp Cloud message webhooks
- outbound text delivery through the Cloud API with 4096-character chunk splitting and shared retry or rate-limit classification
- restart-safe resume handling through the shared bridge delivery broker

## Build

From the repository root:

```bash
go build -o ./extensions/bridges/whatsapp/bin/whatsapp ./extensions/bridges/whatsapp
```

## Install

Build the binary first, then install the extension directory:

```bash
agh extension install ./extensions/bridges/whatsapp
```

## Provider Config

The bridge instance `provider_config` JSON object currently supports:

```json
{
  "api_base_url": "https://graph.facebook.com",
  "api_version": "v21.0",
  "phone_number_id": "1234567890",
  "webhook": {
    "listen_addr": "127.0.0.1:8080",
    "path": "/whatsapp/brg-main"
  },
  "dm": {
    "allow_user_ids": ["15551234567"],
    "allow_usernames": ["alice example"],
    "paired_user_ids": ["15551234567"],
    "paired_usernames": ["alice example"]
  },
  "batching": {
    "delay_ms": 0,
    "split_delay_ms": 0,
    "split_threshold": 0
  }
}
```

Notes:

- `access_token`, `app_secret`, and `verify_token` are required through bridge secret bindings.
- `provider_config.phone_number_id` is required per bridge instance because the runtime multiplexes multiple business numbers behind one provider process.
- `AGH_BRIDGE_WHATSAPP_LISTEN_ADDR` and `AGH_BRIDGE_WHATSAPP_API_BASE_URL` can provide process-level defaults for local development and integration tests.
- Direct-message enforcement uses the bridge instance `dm_policy` plus the provider-config allowlist or paired-user fields.
- WhatsApp Cloud API does not support bridge-level delete semantics and the provider reports those requests as permanent unsupported operations.
