# TC-FUNC-014: Stable ID generation is deterministic and collision-resistant

**Priority:** P1 (High)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `stableID()`

## Objective

Validate that the stable ID function produces deterministic, unique, prefix-tagged IDs based on input parts.

## Preconditions

None (pure function)

## Test Steps

1. Call `stableID("act", "ext-a", "bundle-1", "profile-1", "global", "")` twice
   **Expected:** Same result both times

2. Call with different extension name
   **Expected:** Different ID

3. Call with different prefix ("act" vs "job")
   **Expected:** Different ID (prefix is not part of hash, but different prefix changes result string)

4. Call with whitespace-padded inputs: `stableID("act", "  ext-a  ", "bundle-1")`
   **Expected:** Same ID as `stableID("act", "ext-a", "bundle-1")` (inputs are trimmed)

5. Verify ID format
   **Expected:** Pattern: `{prefix}_{16_hex_chars}` (SHA256 first 8 bytes = 16 hex chars)

6. Call with empty parts
   **Expected:** Produces valid ID (empty strings after trim are valid hash inputs)

## Edge Cases

- Very long input strings → still produces 16-char hex hash
- Unicode input → hashed as UTF-8 bytes
- Parts separated by newline in hash input → different from parts without newline
