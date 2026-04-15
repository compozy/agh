## TC-SEC-004: Webhook Body Size Enforcement

**Priority:** P0
**Type:** Security
**Risk Level:** Critical
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-15

---

### Objective

Verify that the 1MB default body size limit is enforced before signature verification and body parsing, preventing memory exhaustion and denial-of-service attacks via oversized webhook payloads.

### Preconditions

- [ ] Bridge adapter runtime is running with at least one provider instance (e.g., Slack)
- [ ] Default body size limit is configured at 1MB (1,048,576 bytes)
- [ ] HTTP client capable of sending large payloads is available
- [ ] Memory monitoring is available (e.g., runtime metrics, `pprof`, or OS-level monitoring)

### Test Steps

1. **Body exactly at 1MB limit**
   - Input: POST to webhook endpoint with a body of exactly 1,048,576 bytes (valid JSON padded with whitespace). Include valid signature.
   - **Expected:** Request accepted (proceeds to signature verification). Body is fully read and processed.

2. **Body 1 byte over 1MB limit**
   - Input: POST to webhook endpoint with a body of 1,048,577 bytes.
   - **Expected:** Request rejected with 413 Payload Too Large. Response returned before reading the full body. No signature verification attempted.

3. **Body significantly over limit (10MB)**
   - Input: POST to webhook endpoint with a 10MB body.
   - **Expected:** 413 Payload Too Large. Connection closed promptly. Server does not allocate 10MB of memory for the request body.

4. **Body significantly over limit (100MB)**
   - Input: POST to webhook endpoint with a 100MB body (streamed slowly).
   - **Expected:** 413 Payload Too Large. Connection terminated early. No memory spike observed on the server (verify via metrics).

5. **Chunked transfer encoding with oversized payload**
   - Input: Send a POST with `Transfer-Encoding: chunked` and stream chunks totaling 2MB. Do not send a `Content-Length` header.
   - **Expected:** Server tracks bytes read from the chunked stream and rejects with 413 once the 1MB threshold is exceeded. The server does not wait for the final chunk.

6. **Content-Length header mismatch (understated)**
   - Input: Send a POST with `Content-Length: 1000` but actually transmit a 2MB body.
   - **Expected:** Server enforces the limit based on actual bytes read, not the Content-Length header. Request rejected with 413 once actual bytes exceed 1MB.

7. **Compressed body (Content-Encoding: gzip) expanding beyond limit**
   - Input: Send a POST with a gzip-compressed body that is 500KB compressed but decompresses to 5MB.
   - **Expected:** If decompression occurs before size check: reject with 413 after decompressed size exceeds limit. If decompression occurs after size check: accept the compressed body (500KB < 1MB) but reject or truncate at decompression. Either behavior is acceptable as long as 5MB is never fully allocated.

8. **Memory allocation verification under size limit attack**
   - Input: Send 100 concurrent requests, each with a 2MB body, to the same webhook endpoint.
   - **Expected:** All rejected with 413. Server memory usage does not spike proportionally to 100 x 2MB. Body reads are terminated early via `io.LimitReader` or equivalent.

9. **Empty body**
   - Input: Send a POST with `Content-Length: 0` and no body.
   - **Expected:** Request passes size validation (0 < 1MB). Proceeds to subsequent validation stages (signature check, content-type check). May fail at later validation but not at size check.

10. **Enforcement ordering: size check before signature verification**
    - Input: Send a POST with a 2MB body and a valid HMAC-SHA256 signature computed over the full 2MB body.
    - **Expected:** 413 Payload Too Large returned. The server never reaches signature verification. This confirms the security pipeline ordering: method -> content-type -> body size -> rate limit -> signature.

### Attack Vectors

- [ ] Memory exhaustion via large payloads flooding the webhook endpoint
- [ ] Slow-loris-style attacks sending oversized bodies byte-by-byte
- [ ] Chunked encoding bypass to avoid Content-Length-based size checks
- [ ] Gzip bomb / decompression bomb expanding small payloads into huge memory allocations
- [ ] Content-Length header spoofing to bypass size limits
- [ ] Concurrent large payload attacks to multiply memory impact

### Related Test Cases

- TC-SEC-001 (Signature verification — must occur after body size check)
- TC-SEC-003 (Method validation — must occur before body size check)
- TC-SEC-008 (Rate limiting — additional layer against volumetric attacks)
- TC-SEC-009 (In-flight concurrency — limits parallel processing of large payloads)
