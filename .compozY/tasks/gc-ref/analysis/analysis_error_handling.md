# GoClaw Error Handling Patterns — Analysis for AGH

## Key Findings

### 1. Layered Error Classification System (HIGH IMPACT)

**Source:** `internal/providers/error_classify.go`

Three-tier classification:

#### FailoverReason Enum

```go
type FailoverReason string
const (
    FailoverAuth          FailoverReason = "auth"
    FailoverAuthPermanent FailoverReason = "auth_permanent"
    FailoverFormat        FailoverReason = "format"
    FailoverRateLimit     FailoverReason = "rate_limit"
    FailoverOverloaded    FailoverReason = "overloaded"
    FailoverBilling       FailoverReason = "billing"
    FailoverTimeout       FailoverReason = "timeout"
    FailoverModelNotFound FailoverReason = "model_not_found"
    FailoverUnknown       FailoverReason = "unknown"
)
```

#### ErrorClassifier Interface

```go
type ErrorClassifier interface {
    Classify(err error, statusCode int, body string) FailoverClassification
}
```

#### DefaultClassifier — HTTP status + body pattern matching:

- `429` → `FailoverRateLimit`
- `401/403` → checks "revoked"/"expired" → `FailoverAuthPermanent` vs `FailoverAuth`
- `402` → `FailoverBilling`
- `529` or "overload" → `FailoverOverloaded`
- Network errors → `FailoverTimeout`

**Key insight:** Separates error detection (what went wrong) from error handling (what to do about it).

---

### 2. User-Facing Error Transformation (MEDIUM IMPACT)

**Source:** `cmd/gateway_errors.go`

`formatAgentError(err error) string` — classification-driven transformation:

1. **Timeout** (checked FIRST — prevents false positives)
2. **Context overflow** (multi-heuristic: "context length exceeded", "too many tokens")
3. **Role/message format** (tool_use_id mismatch, roles must alternate)
4. **Rate limit** (429, quota exceeded)
5. **Overloaded** (service temporarily busy)
6. **Billing** (insufficient credits)
7. **Auth** (invalid API key)
8. **Model config** (invalid model)
9. **Generic** (log full error, show safe message)

Uses `containsAny(lower, "pattern1", "pattern2")` helper for robust substring matching.

---

### 3. HTTPError Custom Type (HIGH IMPACT)

**Source:** `internal/providers/retry.go`

```go
type HTTPError struct {
    Status     int
    Body       string
    RetryAfter time.Duration // parsed from Retry-After header
}

func (e *HTTPError) Error() string {
    return fmt.Sprintf("HTTP %d: %s", e.Status, e.Body)
}
```

Used with `errors.As()` — allows classification code to extract HTTP details without string parsing.

---

### 4. Retry with Exponential Backoff (HIGH IMPACT)

**Source:** `internal/providers/retry.go`

```go
type RetryConfig struct {
    Attempts int           // 3 default
    MinDelay time.Duration // 300ms default
    MaxDelay time.Duration // 30s default
    Jitter   float64       // 0.1 default (±10%)
}

func RetryDo[T any](ctx context.Context, cfg RetryConfig, fn func() (T, error)) (T, error)

func IsRetryableError(err error, statusCode int) bool {
    // HTTP: 429, 500, 502, 503, 504
    // Network: connection reset, broken pipe, EOF, timeout
    // Non-retryable: 4xx (except 429), auth errors
}
```

Respects `Retry-After` header from HTTP responses.

---

### 5. Protocol-Level Error Responses (MEDIUM IMPACT)

**Source:** `pkg/protocol/errors.go`

```go
type ErrorShape struct {
    Code         string `json:"code"`
    Message      string `json:"message"`
    Details      any    `json:"details,omitempty"`
    Retryable    bool   `json:"retryable,omitempty"`
    RetryAfterMs int    `json:"retryAfterMs,omitempty"`
}

const (
    ErrInvalidRequest    = "INVALID_REQUEST"
    ErrUnavailable       = "UNAVAILABLE"
    ErrUnauthorized      = "UNAUTHORIZED"
    ErrNotFound          = "NOT_FOUND"
    ErrAlreadyExists     = "ALREADY_EXISTS"
    ErrResourceExhausted = "RESOURCE_EXHAUSTED"
    ErrInternal          = "INTERNAL"
)
```

---

### 6. Background Error Alerting (LOW IMPACT for now)

**Source:** `internal/bgalert/report.go`

Only specific error types trigger user alerts:

```go
var alertableReasons = map[providers.FailoverReason]bool{
    providers.FailoverAuth:          true,
    providers.FailoverAuthPermanent: true,
    providers.FailoverBilling:       true,
    providers.FailoverModelNotFound: true,
}
```

Separates operational errors (retry silently) from user-visible errors (alert via WS).

---

### 7. Sentinel Errors with errors.Is()

```go
var ErrTaskNotFound    = errors.New("task not found")
var ErrCronJobNotFound = errors.New("cron job not found")
var ErrInvalidTenant   = errors.New("tenant_id cannot be nil")
```

Checked with `errors.Is()`, never string comparison.

---

## Utility Snippets

### containsAny Helper

```go
func containsAny(s string, substrs ...string) bool {
    for _, sub := range substrs {
        if strings.Contains(s, sub) { return true }
    }
    return false
}
```

### Exponential Backoff with Jitter

```go
func computeRetryDelay(cfg Config, attempt int) time.Duration {
    baseDelay := cfg.MinDelay * time.Duration(math.Pow(2, float64(attempt-1)))
    if baseDelay > cfg.MaxDelay { baseDelay = cfg.MaxDelay }
    jitterRange := time.Duration(float64(baseDelay) * cfg.Jitter)
    offset := time.Duration(rand.Int63n(int64(2*jitterRange))) - jitterRange
    return baseDelay + offset
}
```

---

## Recommended Adaptations for AGH

| Priority | Component                             | Effort | Impact                    |
| -------- | ------------------------------------- | ------ | ------------------------- |
| P0       | Sentinel errors + `errors.Is()`       | Low    | Foundation                |
| P0       | ErrorShape protocol type              | Low    | Client-server error comms |
| P0       | Error classification enum (ErrorKind) | Low    | Retry/alert decision      |
| P1       | HTTPError custom type + `errors.As()` | Low    | Clean error inspection    |
| P1       | User-facing error formatter           | Medium | Better UX                 |
| P1       | Retry config + `IsRetryable()`        | Medium | Resilience                |
| P2       | `RetryDo[T]` generic executor         | Medium | Reusable retry            |
| P3       | Background error alerting             | Medium | Only if async workers     |
