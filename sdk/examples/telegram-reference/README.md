# Telegram Reference Adapter

`telegram-reference` is the Go reference adapter for AGH's negotiated channel runtime.

It demonstrates:

- launch-time channel metadata and bound secret injection through `initialize.runtime.channel`
- inbound platform normalization through `channels/messages/ingest`
- outbound negotiated delivery through `channels/deliver`
- adapter-driven instance state reporting through `channels/instances/report_state`
- restart-safe delivery markers that the conformance harness can validate

This example is intentionally fake-platform and CI-safe. Instead of talking to the real Telegram API, it tails a JSONL file of Telegram-like updates and writes JSON/JSONL markers that the integration harness reads back.

## Build

From the repository root:

```bash
go build -o ./sdk/examples/telegram-reference/bin/telegram-reference ./sdk/examples/telegram-reference
```

Or from this directory:

```bash
mkdir -p bin
go build -o ./bin/telegram-reference .
```

## Install

Build the binary first, then install the extension directory:

```bash
agh extension install ./sdk/examples/telegram-reference
```

## Manifest Summary

- Capability: `channel.adapter`
- Host API actions: `channels/messages/ingest`, `channels/instances/get`, `channels/instances/report_state`
- Security grants: `channel.read`, `channel.write`
- Extension service: `channels/deliver`

## Fake Platform Contract

The runtime watches the file named by `AGH_CHANNEL_ADAPTER_UPDATES_PATH`. Each non-empty line must be one Telegram-like update JSON object. The minimal supported shape is:

```json
{
  "update_id": 1001,
  "message": {
    "message_id": 9,
    "date": 1775866800,
    "chat": { "id": 42, "type": "private" },
    "from": { "id": 7, "username": "alice", "first_name": "Alice" },
    "text": "hello"
  }
}
```

## Marker Environment

The adapter reads these optional environment variables. They are used by the conformance harness and can also help extension authors debug runtime behavior:

- `AGH_CHANNEL_ADAPTER_HANDSHAKE_PATH`: writes the initialize request/response marker as JSON.
- `AGH_CHANNEL_ADAPTER_INSTANCE_PATH`: writes the resolved `channels/instances/get` result as JSON.
- `AGH_CHANNEL_ADAPTER_STATE_PATH`: appends one JSON line per reported channel status.
- `AGH_CHANNEL_ADAPTER_DELIVERY_PATH`: appends one JSON line per `channels/deliver` request, including the returned ack when available.
- `AGH_CHANNEL_ADAPTER_INGEST_PATH`: appends one JSON line per fake inbound update ingest attempt.
- `AGH_CHANNEL_ADAPTER_UPDATES_PATH`: JSONL file polled for fake inbound Telegram updates.
- `AGH_CHANNEL_ADAPTER_STARTS_PATH`: appends one line per runtime process start.
- `AGH_CHANNEL_ADAPTER_SHUTDOWN_PATH`: appends one line when the daemon sends `shutdown`.
- `AGH_CHANNEL_ADAPTER_CRASH_ONCE_PATH`: if set and the file does not exist yet, the runtime exits on its first outbound delivery after writing the request marker. The broker should then resume delivery after restart.

## Bound Credentials

The adapter reads only `initialize.runtime.channel.bound_secrets`. For the ready path, it expects a `bot_token` binding. If that binding is missing, it reports `auth_required` and never attempts any arbitrary runtime secret lookup.
