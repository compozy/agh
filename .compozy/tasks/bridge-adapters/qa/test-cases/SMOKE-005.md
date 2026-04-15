## SMOKE-005: Delivery Pipeline Completes START to FINAL

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-15

---

### Objective

Verify the delivery broker can project a delivery through the full START→FINAL lifecycle and receive a valid DeliveryAck.

### Preconditions

- [ ] Delivery broker initialized with a recording transport
- [ ] At least one active route worker

### Test Steps

1. **Enqueue a delivery with START event containing text content**
   - **Expected:** Route worker picks up delivery, transport receives START event

2. **Send FINAL event completing the delivery**
   - **Expected:** Transport receives FINAL event with isFinal=true

3. **Verify DeliveryAck returned with DeliveryID and RemoteMessageID**
   - **Expected:** Ack fields populated, delivery marked complete in broker

### Related Test Cases

- TC-FUNC-009, TC-FUNC-010, TC-INT-004
