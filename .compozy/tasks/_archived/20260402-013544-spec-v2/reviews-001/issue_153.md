---
status: resolved
file: web/src/lib/stores/sessions.ts
line: 60
severity: low
author: claude-reviewer
---

# Issue 153: Sessions store polling continues even when tab is backgrounded



## Review Comment

The sessions store uses `setInterval` for polling at a fixed 5-second interval:

```typescript
start() {
    void refresh();
    if (poller !== null) {
        window.clearInterval(poller);
    }
    poller = window.setInterval(() => {
        void refresh();
    }, pollMs);
}
```

This polling continues even when the browser tab is backgrounded or the page is not visible. While modern browsers throttle `setInterval` in background tabs, the HTTP requests still fire, consuming bandwidth and server resources unnecessarily.

**Suggested fix**: Use the Page Visibility API to pause polling when the tab is hidden:

```typescript
const handleVisibility = () => {
    if (document.hidden) {
        if (poller !== null) window.clearInterval(poller);
    } else {
        void refresh();
        poller = window.setInterval(() => void refresh(), pollMs);
    }
};
document.addEventListener('visibilitychange', handleVisibility);
```

## Triage

- Decision: `valid`
- Notes:
  - The sessions store keeps its polling interval active regardless of page visibility, so it continues issuing background requests when the dashboard tab is hidden.
  - That is an actual efficiency bug with avoidable network churn, not just an optional enhancement.
  - The fix should pause polling while hidden and resume with an immediate refresh when the tab becomes visible again.
  - Resolution: the sessions store now pauses polling while `document.hidden` is true, resumes on `visibilitychange`, and has coverage in `web/src/lib/stores/sessions.spec.ts`.
