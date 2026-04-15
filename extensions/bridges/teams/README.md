# Teams Bridge Provider

Production Microsoft Teams bridge provider built on `internal/bridgesdk`.

## Secrets

- `app_id`: Microsoft bot application ID.
- `app_password`: Microsoft bot client secret.
- `app_tenant_id`: optional single-tenant pinning for outbound Bot Framework token acquisition and DM creation.

## Provider Config

Provider config is stored per bridge instance in `provider_config`:

```json
{
  "service_url": "https://smba.trafficmanager.net/teams/",
  "webhook": {
    "listen_addr": "127.0.0.1:0",
    "path": "/teams/brg-example"
  },
  "auth": {
    "openid_metadata_url": "https://login.botframework.com/v1/.well-known/openidconfiguration",
    "token_url": "https://login.microsoftonline.com/botframework.com/oauth2/v2.0/token"
  },
  "batching": {
    "delay_ms": 50,
    "split_delay_ms": 50,
    "split_threshold": 2
  },
  "dm": {
    "allow_user_ids": ["29:example"],
    "paired_user_ids": ["29:paired"]
  }
}
```

`service_url` should usually be learned from inbound activities. Configure it only as a fallback for proactive delivery or tests.

## Scope

Bridge v1 support in this provider includes:

- inbound Teams message activities
- inbound adaptive-card/message-submit actions
- inbound message reactions
- outbound post, edit, and delete delivery
- tenant-aware proactive DM creation when only a user ID is available

Task modules, modal lifecycle flows, and richer Teams UI parity stay out of scope for v1.
