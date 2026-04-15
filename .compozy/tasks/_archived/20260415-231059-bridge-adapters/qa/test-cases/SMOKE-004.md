## SMOKE-004: Inbound Message Ingestion Through Host API

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-15

---

### Objective

Verify a provider can call Host API `messages/ingest` with a valid inbound message envelope and the daemon accepts it.

### Preconditions

- [ ] Bridge SDK HostAPIClient available
- [ ] At least one bridge instance in ready state

### Test Steps

1. **Construct an InboundMessageEnvelope with bridge_instance_id, peer_id, text content**
   - **Expected:** Envelope valid

2. **Call HostAPIClient.Ingest() with the envelope**
   - **Expected:** No error returned, daemon ingests the message

3. **Verify the daemon received the event with correct routing dimensions**
   - **Expected:** Event routed with matching bridge_instance_id and peer_id

### Related Test Cases

- TC-FUNC-007, TC-INT-002
