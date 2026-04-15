## TC-INT-002: Webhook Ingress to Daemon Ingest Flow

**Priority:** P0
**Type:** Integration
**Systems:** bridgesdk.WebhookGuard, bridgesdk.HostAPIClient, extension.HostAPI (bridges/messages/ingest), bridges.InboundMessageEnvelope, bridges.RoutingKey, store/globaldb
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-15

---

### Objective

Validate the full inbound path: an HTTP webhook request arrives at the provider's webhook endpoint, passes the ingress guard pipeline (method check, content-type check, body limit, rate limiter, signature verification), the provider maps the platform-specific payload to a normalized `InboundMessageEnvelope`, calls Host API `bridges/messages/ingest`, the daemon resolves or creates a route, and returns a `BridgesMessagesIngestResult` with the target session ID.

### Preconditions

- [ ] Provider runtime is initialized with at least 1 enabled bridge instance (`brg-wh-1`, scope=global, platform=telegram)
- [ ] WebhookGuardConfig is configured with: AllowedMethods=["POST"], AllowedContentTypes=["application/json"], MaxBodyBytes=1MB, VerifySignature function bound to the provider's HMAC secret
- [ ] Rate limiter set to 100 requests/minute per source IP
- [ ] InFlightLimiter set to 10 concurrent requests
- [ ] globaldb bridge_routes table is empty (first message creates a new route)

### Test Steps

1. **Send a valid webhook POST with correct signature**
   - Input: HTTP POST to `http://<listen_addr>/<platform>` with Content-Type `application/json`, valid HMAC signature header, body containing a platform-specific message event (e.g., Telegram Update with `message.text="hello"`)
   - **Expected:** HTTP 200 response; provider's WebhookHandler is invoked with `WebhookRequest.Body` containing the raw payload and `ReceivedAt` set to a recent timestamp

2. **Verify provider maps payload to InboundMessageEnvelope**
   - Input: Capture the envelope constructed by the provider's webhook handler
   - **Expected:** `bridge_instance_id` = `brg-wh-1`; `scope` = `global`; `peer_id` = extracted sender ID; `platform_message_id` = extracted message ID; `event_family` = `message`; `content.text` = `hello`; `idempotency_key` is non-empty and deterministic for the same platform message

3. **Verify provider calls Host API bridges/messages/ingest**
   - Input: Capture the JSON-RPC call issued by the HostAPIClient
   - **Expected:** Method = `bridges/messages/ingest`; params match the envelope from step 2; `received_at` is a valid RFC3339 timestamp

4. **Verify daemon processes the ingest and creates a route**
   - Input: Inspect the `BridgesMessagesIngestResult` returned to the provider
   - **Expected:** `session_id` is non-empty; `route_created` = `true`; `routing_key.scope` = `global`; `routing_key.bridge_instance_id` = `brg-wh-1`; `routing_key.peer_id` = extracted sender ID

5. **Send a second message from the same sender**
   - Input: Same webhook POST with a new `platform_message_id` and different text
   - **Expected:** `route_created` = `false` (existing route reused); `session_id` matches step 4

6. **Verify idempotency: replay the first message**
   - Input: Resend the exact same webhook POST from step 1 (same body, same signature)
   - **Expected:** The daemon deduplicates via `IngestDedupRecord`; either returns the same result without creating a duplicate event or returns an appropriate dedup response

### Data Validation

| Field                                 | Source Value                                       | Transformed Value                                  | Status |
| ------------------------------------- | -------------------------------------------------- | -------------------------------------------------- | ------ |
| HTTP request body                     | Platform-specific JSON                             | WebhookRequest.Body (raw bytes)                    |        |
| Telegram update.message.from.id       | `12345`                                            | InboundMessageEnvelope.PeerID = `12345`            |        |
| Telegram update.message.message_id    | `67890`                                            | InboundMessageEnvelope.PlatformMessageID = `67890` |        |
| Telegram update.message.text          | `hello`                                            | InboundMessageEnvelope.Content.Text = `hello`      |        |
| Computed HMAC                         | sha256(secret, body)                               | Signature header value                             |        |
| InboundMessageEnvelope.IdempotencyKey | deterministic hash of platform+instance+message_id | IngestDedupRecord.IdempotencyKey                   |        |

### Error Scenarios

- [ ] Wrong HTTP method (GET instead of POST): returns 405 Method Not Allowed
- [ ] Wrong Content-Type (text/plain): returns 415 Unsupported Media Type
- [ ] Body exceeds MaxBodyBytes: returns 413 Request Entity Too Large
- [ ] Invalid HMAC signature: returns 401 Unauthorized
- [ ] Rate limiter exceeded: returns 429 Too Many Requests
- [ ] InFlight limiter saturated: returns 503 Service Unavailable
- [ ] bridge_instance_id not found in daemon registry: Host API returns error, provider returns 500
- [ ] Malformed JSON body: provider mapping fails, returns 400 or 500 with error detail
- [ ] InboundMessageEnvelope fails validation (e.g., empty idempotency_key): Host API returns validation error

### Related Test Cases

- TC-INT-001 (provider must be launched with instances before webhooks work)
- TC-INT-003 (multi-instance routing isolation for distinct peer/thread combinations)
- TC-INT-006 (auth_required instance should reject ingest attempts)
