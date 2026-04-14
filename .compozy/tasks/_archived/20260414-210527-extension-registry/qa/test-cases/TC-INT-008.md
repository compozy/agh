# TC-INT-008: GitHub Rate Limit Handling

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Integration |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/github/client.go` |

## Objective

Validate that the GitHub adapter correctly handles rate limit headers and provides clear guidance.

## Preconditions

- Mock server returning `X-RateLimit-Remaining` headers.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Response with `X-RateLimit-Remaining: 5` | **Expected:** Warning logged about approaching rate limit. Operation succeeds. |
| 2 | Response with `X-RateLimit-Remaining: 0` and HTTP 403 | **Expected:** Error message suggesting `GITHUB_TOKEN` environment variable. |
| 3 | Response with `X-RateLimit-Remaining: 100` | **Expected:** No warning. Normal operation. |

## Edge Cases

- Missing rate limit headers: should not panic, treat as unlimited.
- `GITHUB_TOKEN` set but invalid: should return 401 error with clear message.
