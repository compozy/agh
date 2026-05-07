# TC-SEC-001: No Secret Material Leaks Across Surfaces

**Priority:** P0
**Type:** Security
**OWASP Category:** A09 (logging) / A02 (cryptographic failures)
**Risk Level:** Critical
**Requirement:** TechSpec SI-9.
**Status:** Not Run

## Objective

Verify API keys, OAuth tokens, secret-shaped env vars, and provider credential material never appear in any source error, log, status payload, SSE event, web-visible payload, or Host API response.

## Preconditions

- [ ] Daemon running; structured logs captured.
- [ ] Source stubs configured to return errors that include API key, OAuth token, env-shaped secrets.
- [ ] Provider env explicitly seeds `OPENAI_API_KEY=sk-test-1234567890abcdef`, `ANTHROPIC_API_KEY=sk-ant-secret`, `OAUTH_REFRESH_TOKEN=oauth.refresh.secret`.

## Test Steps

1. **Trigger refresh failures with seeded errors for `models.dev`, live providers, extension source.**
2. **Capture logs (stdout + structured), HTTP/UDS status responses, CLI output, Host API response, web `network` traffic from Settings > Providers, SSE events.**
3. **Grep all captured payloads for the seeded secret strings.**
   - **Expected:** Zero matches.
4. **Reduce redaction helper to no-op (test harness override) and re-run.**
   - **Expected:** Projection-time redaction still catches secrets; defense-in-depth confirmed.
5. **Restore redaction helper; introduce a new secret-looking string in error.**
   - **Expected:** Redacted summary remains readable but obfuscates secret-shaped substrings.

## Audit Coverage

- C11 disruption probe, C14.
- SI-9.

## Pass Criteria

- No secret leak across any surface.

## Failure Criteria

- Secret string appears in any captured surface.
- Redaction toggleable from outside redact helper.
