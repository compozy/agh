## TC-SEC-003: Webhook Method Validation

**Priority:** P0
**Type:** Security
**Risk Level:** Critical
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective
Verify that webhook endpoints reject all HTTP methods except POST (and GET for Telegram/WhatsApp verification endpoints) before any body parsing, signature verification, or business logic executes, preventing method-based bypass attacks.

### Preconditions
- [ ] Bridge adapter runtime is running with at least one provider instance registered (e.g., Slack, GitHub, Discord)
- [ ] Webhook endpoints are accessible
- [ ] HTTP client capable of sending arbitrary HTTP methods is available

### Test Steps

1. **GET request to POST-only webhook endpoint**
   - Input: Send `GET /webhooks/slack/{instance_id}` with no body.
   - **Expected:** 405 Method Not Allowed returned. Response includes `Allow: POST` header. No signature verification attempted. No event processing.

2. **PUT request to webhook endpoint**
   - Input: Send `PUT /webhooks/github/{instance_id}` with a valid JSON body and valid signature headers.
   - **Expected:** 405 Method Not Allowed. The valid signature is irrelevant — method check occurs first.

3. **DELETE request to webhook endpoint**
   - Input: Send `DELETE /webhooks/discord/{instance_id}`.
   - **Expected:** 405 Method Not Allowed.

4. **PATCH request to webhook endpoint**
   - Input: Send `PATCH /webhooks/linear/{instance_id}` with `{"update":"malicious"}`.
   - **Expected:** 405 Method Not Allowed.

5. **OPTIONS request (CORS preflight)**
   - Input: Send `OPTIONS /webhooks/slack/{instance_id}` with CORS headers.
   - **Expected:** Either 405 or a valid CORS preflight response (if CORS is configured). No body parsing or signature verification occurs.

6. **HEAD request to webhook endpoint**
   - Input: Send `HEAD /webhooks/github/{instance_id}`.
   - **Expected:** 405 Method Not Allowed. No body processing.

7. **Custom/non-standard HTTP method**
   - Input: Send `PROPFIND /webhooks/slack/{instance_id}` (WebDAV method).
   - **Expected:** 405 Method Not Allowed. Non-standard methods are not routed to webhook handlers.

8. **Verify ordering: method validation before body read**
   - Input: Send `PUT /webhooks/slack/{instance_id}` with a 500KB body and invalid Content-Type. Measure whether the response is immediate.
   - **Expected:** 405 returned without reading the request body. Response latency is negligible (body not consumed).

9. **POST request to webhook endpoint (positive control)**
   - Input: Send `POST /webhooks/slack/{instance_id}` with valid Content-Type and body (signature may be invalid for this test).
   - **Expected:** Request proceeds past method validation. Rejected later at signature verification (401/403), confirming POST is the only accepted method for this stage.

10. **GET request to Telegram verification endpoint (if applicable)**
    - Input: Send `GET /webhooks/telegram/{instance_id}` with Telegram's verification query parameters.
    - **Expected:** If Telegram verification is handled via GET, the request is accepted and processed. Otherwise, 405.

### Attack Vectors
- [ ] Method confusion attacks using PUT/PATCH to bypass POST-only security middleware
- [ ] CSRF via GET requests (browsers may send cross-origin GET requests without CORS preflight)
- [ ] WebDAV method injection (PROPFIND, MOVE, COPY) to probe for misconfigured routers
- [ ] HEAD requests to probe endpoint existence without triggering full processing
- [ ] Method override headers (`X-HTTP-Method-Override`) to bypass method restrictions

### Related Test Cases
- TC-SEC-001 (Signature verification — occurs after method validation)
- TC-SEC-004 (Body size limits — occurs after method validation)
- TC-SEC-008 (Rate limiting — may interact with method validation ordering)
