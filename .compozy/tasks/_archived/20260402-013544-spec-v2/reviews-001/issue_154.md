---
status: resolved
file: web/src/lib/utils/shortcuts.ts
line: 20
severity: medium
author: claude-reviewer
---

# Issue 154: Keyboard shortcuts fire even when user is typing in an input field



## Review Comment

The `resolveDashboardShortcut` function receives an `isTypingTarget` context field but does not use it to suppress shortcuts when the user is typing in an input field:

```typescript
export function resolveDashboardShortcut(
    context: DashboardShortcutContext
): DashboardShortcutAction | null {
    if (context.key >= '1' && context.key <= '9') {
        return {
            type: 'set-zoom',
            zoom: Number(context.key) / 10
        };
    }

    switch (context.key) {
        case '[':
            return { type: 'toggle-sidebar' };
        case '0':
            return { type: 'fit-all' };
        case '/':
            return { type: 'open-search' };
        case '?':
            return { type: 'open-shortcuts' };
        case 'Escape':
            return {
                type: 'escape',
                clearSelection: !context.isTypingTarget
            };
        case 'j':
            return { type: 'move-selection', direction: 1 };
        case 'k':
            return { type: 'move-selection', direction: -1 };
        ...
    }
}
```

The `isTypingTarget` field is only used for the Escape action's `clearSelection` property. But when a user types `[`, `0`, `/`, `j`, `k`, `?`, or any digit `1`-`9` in the search input, these characters will trigger dashboard shortcuts instead of being typed. For example, typing a `0` in the search field will trigger "fit all" instead of entering the character, and digits `1`-`9` will change the zoom level.

In the `App.svelte` handler, `event.preventDefault()` is called for all resolved actions, which means the character won't even be typed in the input field.

**Suggested fix**: Add an early return in `resolveDashboardShortcut` when `isTypingTarget` is true, except for specific keys like Escape and Enter that should work even in input fields:

```typescript
if (context.isTypingTarget && !['Escape', 'Enter'].includes(context.key)) {
    return null;
}
```

## Triage

- Decision: `valid`
- Notes:
  - `resolveDashboardShortcut` receives `isTypingTarget` but still resolves character shortcuts like `0`, `/`, `?`, `j`, `k`, and digits while focus is inside an input.
  - `App.svelte` prevents default for every resolved shortcut, so these keys are actively stolen from the user instead of being typed.
  - This is a concrete input-handling bug and should be fixed by suppressing most shortcuts while typing, while still allowing deliberate exceptions such as `Escape` and relevant `Enter` behavior.
  - Resolution: printable shortcuts are now suppressed for typing targets while preserving `Escape`/`Enter`, with regression coverage in `web/src/lib/utils/shortcuts.spec.ts`.
