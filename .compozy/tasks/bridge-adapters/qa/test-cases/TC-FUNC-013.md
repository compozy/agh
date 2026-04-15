## TC-FUNC-013: Error Classification Mapping

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective
Validate that the `ClassifyError` function in `internal/bridgesdk/errors.go` correctly maps representative provider failures (HTTP status codes, typed errors, context errors, network errors, and text-based heuristics) into the five error classes: `auth`, `rate_limit`, `timeout`, `transient`, `permanent`, and that `Recovery()` returns the correct recovery decision for each class.

### Preconditions
- [ ] `internal/bridgesdk` package is compiled and testable
- [ ] `ClassifyError` and `ClassifiedError.Recovery()` functions are available
- [ ] Understanding of the recovery decision model: `RecoveryDecision{Retry, RetryAfter, Status, Degradation}`

### Test Steps
1. **HTTP 401 Unauthorized -> auth**
   - Input: `&HTTPError{StatusCode: 401, Message: "invalid token"}`
   - **Expected:**
     - `ClassifiedError.Class` = `ErrorClassAuth` (`"auth"`)
     - `Recovery().Retry` = `false`
     - `Recovery().Status` = `BridgeStatusAuthRequired`
     - `Recovery().Degradation.Reason` = `BridgeDegradationReasonAuthFailed`

2. **HTTP 403 Forbidden -> auth**
   - Input: `&HTTPError{StatusCode: 403, Message: "forbidden"}`
   - **Expected:** `Class` = `"auth"`, same recovery as 401

3. **HTTP 429 Too Many Requests -> rate_limit**
   - Input: `&HTTPError{StatusCode: 429, Message: "rate limited", RetryAfter: 30 * time.Second}`
   - **Expected:**
     - `Class` = `ErrorClassRateLimit` (`"rate_limit"`)
     - `Recovery().Retry` = `true`
     - `Recovery().RetryAfter` = `30s`
     - `Recovery().Status` = `BridgeStatusDegraded`
     - `Recovery().Degradation.Reason` = `BridgeDegradationReasonRateLimited`

4. **HTTP 408 Request Timeout -> timeout**
   - Input: `&HTTPError{StatusCode: 408, Message: "request timeout"}`
   - **Expected:**
     - `Class` = `ErrorClassTimeout` (`"timeout"`)
     - `Recovery().Retry` = `true`
     - `Recovery().Status` = `BridgeStatusDegraded`
     - `Recovery().Degradation.Reason` = `BridgeDegradationReasonProviderTimeout`

5. **HTTP 504 Gateway Timeout -> timeout**
   - Input: `&HTTPError{StatusCode: 504, Message: "gateway timeout"}`
   - **Expected:** `Class` = `"timeout"`

6. **HTTP 500 Internal Server Error -> transient**
   - Input: `&HTTPError{StatusCode: 500, Message: "internal error"}`
   - **Expected:**
     - `Class` = `ErrorClassTransient` (`"transient"`)
     - `Recovery().Retry` = `true`
     - `Recovery().Status` = `BridgeStatusDegraded`
     - `Recovery().Degradation` is `nil` (transient has no structured reason)

7. **HTTP 502 Bad Gateway -> transient**
   - Input: `&HTTPError{StatusCode: 502, Message: "bad gateway"}`
   - **Expected:** `Class` = `"transient"`

8. **HTTP 503 Service Unavailable -> transient**
   - Input: `&HTTPError{StatusCode: 503, Message: "service unavailable"}`
   - **Expected:** `Class` = `"transient"`

9. **HTTP 404 Not Found -> permanent**
   - Input: `&HTTPError{StatusCode: 404, Message: "not found"}`
   - **Expected:**
     - `Class` = `ErrorClassPermanent` (`"permanent"`)
     - `Recovery().Retry` = `false`
     - `Recovery().Status` = `BridgeStatusError`
     - `Recovery().Degradation` is `nil`

10. **Typed AuthError -> auth**
    - Input: `&AuthError{Err: errors.New("oauth token expired")}`
    - **Expected:** `Class` = `"auth"`

11. **Typed RateLimitError -> rate_limit**
    - Input: `&RateLimitError{Err: errors.New("too many requests"), RetryAfter: 5 * time.Second}`
    - **Expected:** `Class` = `"rate_limit"`, `RetryAfter` = `5s`

12. **Typed TransientError -> transient**
    - Input: `&TransientError{Err: errors.New("temporary failure")}`
    - **Expected:** `Class` = `"transient"`

13. **Typed PermanentError -> permanent**
    - Input: `&PermanentError{Err: errors.New("channel deleted")}`
    - **Expected:** `Class` = `"permanent"`

14. **context.DeadlineExceeded -> timeout**
    - Input: `context.DeadlineExceeded`
    - **Expected:** `Class` = `"timeout"`

15. **net.Error with Timeout() -> timeout**
    - Input: A `net.Error` where `Timeout()` returns `true`
    - **Expected:** `Class` = `"timeout"`

16. **net.Error without Timeout() -> transient**
    - Input: A `net.Error` where `Timeout()` returns `false`
    - **Expected:** `Class` = `"transient"`

17. **Text heuristic: "unauthorized" -> auth**
    - Input: `errors.New("unauthorized access")`
    - **Expected:** `Class` = `"auth"` (text-based fallback)

18. **Text heuristic: "rate limit" -> rate_limit**
    - Input: `errors.New("hit rate limit")`
    - **Expected:** `Class` = `"rate_limit"`

19. **Text heuristic: unknown error -> permanent**
    - Input: `errors.New("something completely unknown")`
    - **Expected:** `Class` = `"permanent"` (default fallback)

20. **nil error -> empty classification**
    - Input: `nil`
    - **Expected:** `ClassifiedError{}` with empty class

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Wrapped HTTPError | `fmt.Errorf("provider: %w", &HTTPError{StatusCode: 429})` | Unwrapped to rate_limit via errors.As |
| Wrapped AuthError | `fmt.Errorf("slack: %w", &AuthError{Err: errors.New("bad token")})` | Unwrapped to auth |
| HTTP 422 Unprocessable Entity | `&HTTPError{StatusCode: 422}` | permanent (default for non-mapped codes) |
| Text with "connection reset" | `errors.New("connection reset by peer")` | transient |
| Text with "broken pipe" | `errors.New("broken pipe")` | transient |
| Text with "eof" | `errors.New("unexpected eof")` | transient |

### Related Test Cases
- TC-FUNC-014 (degradation reporting uses classified errors)
- TC-FUNC-015 (recovery from rate_limit)
- TC-FUNC-005 (status transitions triggered by error classes)
