## TC-INT-005: Delivery Recovery After Provider Restart

**Priority:** P0
**Type:** Integration
**Systems:** bridges.Broker, bridges.DeliverySnapshot, bridges.DeliveryResumeState, bridgesdk.Runtime, extension.Manager (restart supervision), bridges.DeliveryTransport, subprocess lifecycle
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective

Validate that an in-flight delivery survives a provider subprocess crash and restart. After the provider restarts and re-initializes, the broker sends a RESUME event containing the `DeliverySnapshot` with the latest accumulated content, and the provider uses the snapshot to continue the delivery to completion. Confirms that the broker's `handleSendFailure` schedules a resume, and the provider's runtime handles `DeliveryEventTypeResume` with `DeliveryResumeState.LatestEventType`.

### Preconditions

- [ ] Provider runtime initialized with 1 bridge instance (`brg-res-1`, scope=global, platform=telegram)
- [ ] A route exists for `brg-res-1` + `peer_id=peer-A` -> `session_id=sess-1`
- [ ] Broker is constructed with `WithDeliveryBrokerRetryDelay(50ms)` for fast test iteration
- [ ] Provider process can be killed and restarted via the extension manager's restart supervision
- [ ] Provider's `DeliveryHandler` tracks whether it received a RESUME event

### Test Steps

1. **Register and start a delivery**
   - Input: Register `PromptDeliveryRegistration` for `brg-res-1`, project `agent_message("Hello ")` and `agent_message("world")` events
   - **Expected:** Broker creates a delivery with `delivery_id`, START event queued

2. **Wait for START delivery to succeed**
   - Input: Wait up to 5s for the broker's transport to deliver the START event
   - **Expected:** Provider receives `DeliveryRequest{event.event_type=start, event.content.text="Hello "}`, returns ack with `remote_message_id=rmsg-1`

3. **Project additional DELTA content**
   - Input: Project `agent_message("! How are ")` event
   - **Expected:** Broker updates the active delivery with cumulative content

4. **Kill the provider subprocess**
   - Input: Terminate the provider process (simulate crash)
   - **Expected:** Extension manager detects process exit; broker's next delivery attempt fails with transport error

5. **Verify broker schedules a RESUME**
   - Input: Inspect broker internal state after send failure
   - **Expected:** The failed delivery has `queuedResume=true`; the route worker has a RESUME queue item prepended

6. **Wait for extension manager to restart the provider**
   - Input: Extension manager's restart supervision triggers; new subprocess spawns; new `initialize` handshake completes
   - **Expected:** Provider runtime re-initializes with the same bridge instances; broker's `SetTransport` is called with the new transport

7. **Verify broker sends RESUME event with snapshot**
   - Input: Wait up to 10s for the RESUME delivery request
   - **Expected:** Provider receives `DeliveryRequest{event.event_type=resume, event.resume.latest_event_type=delta, event.content.text="Hello world! How are "}` with a non-nil `snapshot` field

8. **Verify the RESUME snapshot contents**
   - Input: Inspect `DeliveryRequest.Snapshot` received by the provider
   - **Expected:** `snapshot.delivery_id` matches; `snapshot.session_id=sess-1`; `snapshot.turn_id=turn-1`; `snapshot.remote_message_id=rmsg-1`; `snapshot.latest_seq` >= 2; `snapshot.current_content.text` matches accumulated text; `snapshot.final=false`

9. **Complete the delivery after RESUME**
   - Input: Project `done` event after RESUME succeeds
   - **Expected:** Broker sends a FINAL event; provider acks with `replace_remote_message_id` or new `remote_message_id`; broker snapshot shows `final=true`, `last_acked_seq >= latest_seq`

### Data Validation

| Field                               | Source Value                           | Transformed Value                  | Status |
| ----------------------------------- | -------------------------------------- | ---------------------------------- | ------ |
| Pre-crash accumulated text          | "Hello world! How are "                | DeliveryEvent(resume).Content.Text |        |
| DeliveryResumeState.LatestEventType | Last projected event type before crash | "delta" or "start"                 |        |
| DeliverySnapshot.RemoteMessageID    | Ack from pre-crash START               | "rmsg-1" preserved across restart  |        |
| DeliverySnapshot.LastAckedSeq       | Seq of last successful ack             | >= 1 (START was acked)             |        |
| DeliverySnapshot.LastSentSeq        | Seq of last sent event                 | >= LastAckedSeq                    |        |
| Post-resume FINAL content           | Full accumulated text                  | DeliveryEvent(final).Content.Text  |        |

### Error Scenarios

- [ ] Provider fails to restart within the restart backoff threshold: delivery remains queued indefinitely until max restarts exceeded
- [ ] RESUME event arrives but provider's initialize handshake has not completed: broker sees transport error and retries again
- [ ] Provider acks RESUME but then crashes again: broker re-schedules another RESUME with updated snapshot
- [ ] DeliveryResumeState.LatestEventType is itself "resume": validation rejects it (RESUME cannot reference RESUME)
- [ ] Snapshot validation fails (e.g., last_acked_seq > last_sent_seq): broker does not send and logs a validation error
- [ ] Session ends (done event projected) while provider is down: broker accumulates FINAL; RESUME carries `final=true`

### Related Test Cases

- TC-INT-004 (normal delivery flow without restart)
- TC-INT-006 (auth degradation during delivery)
- TC-INT-012 (conformance harness validates restart-recovery target)
