# TC-SEC-007: Oversized Record Payload Rejected

**Priority:** P1
**Type:** Security
**Package:** internal/resources
**Related Tasks:** 01, 02

## Objective

Validate that `PutRaw` rejects records whose `spec_json` payload exceeds the configured `MaxBytes` limit for the record's kind. Rejection must occur before the payload is persisted to the store.

## Preconditions

- The resource runtime is configured with `MaxBytes` limits per kind (e.g., `tool` kind has `MaxBytes=64KB`, `prompt.template` has `MaxBytes=256KB`).
- An operator or extension session is available to submit records.
- The resource store is empty or in a known state.

## Test Steps

1. Submit a `PutRaw` request with a `tool` record whose `spec_json` is exactly at the `MaxBytes` limit (64KB).
   **Expected:** The record is accepted and persisted successfully. Boundary value is within the allowed range.

2. Submit a `PutRaw` request with a `tool` record whose `spec_json` is 1 byte over the `MaxBytes` limit (64KB + 1).
   **Expected:** The request is rejected with 413 Payload Too Large. The error message includes the kind, the limit, and the actual size.

3. Verify the oversized record from step 2 was not persisted.
   **Expected:** A subsequent `resources/get` for the rejected record's `(kind, id)` returns 404. No partial data was written.

4. Submit a `PutRaw` request with a `prompt.template` record whose `spec_json` is 200KB (within its 256KB limit).
   **Expected:** The record is accepted. Different kinds can have different MaxBytes limits.

5. Submit a `PutRaw` request with a record whose `spec_json` is 0 bytes (empty).
   **Expected:** The request is either accepted (if empty payloads are valid) or rejected with a validation error -- not a 413. Empty payloads are a validation concern, not a size concern.

## Edge Cases

- Payload exactly at the limit but with additional metadata fields that push total request size over a reasonable bound.
- Payload contains highly compressed data that would expand significantly if decompressed (zip bomb pattern).
- Payload size is checked against the raw JSON bytes, not the deserialized in-memory size.
- Concurrent submissions of multiple oversized records to verify that rejection is consistent under load.
- Kind with no configured MaxBytes limit -- verify a sensible global default applies rather than unlimited.

## Threat Model

This test prevents **resource exhaustion via payload inflation**. Without size limits, a malicious extension or operator could submit arbitrarily large records that consume disk space in the SQLite store, exhaust memory during deserialization, or degrade query performance. The per-kind MaxBytes configuration allows fine-grained control -- tool definitions are expected to be small, while prompt templates may legitimately be larger. Enforcing limits before persistence ensures that the store never contains oversized records that could cause downstream failures when other components read them.
