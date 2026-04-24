---
status: resolved
file: web/src/systems/network/components/network-workspace-shell.tsx
line: 244
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIen,comment:PRRC_kwDOR5y4QM66CAlP
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Stop keyboard events from the star button from reaching the row handler.**

Line 182 makes the whole row react to `Enter`/`Space`, but the nested star button only stops click propagation. Keyboard activation on the star control will also select the room, and `Space` can even suppress the button action.

<details>
<summary>🔧 Minimal fix</summary>

```diff
         <button
           aria-label={item.isStarred ? "Unstar channel" : "Star channel"}
           className="mt-0.5 rounded-md p-1 text-[color:var(--color-text-tertiary)] transition-colors hover:bg-[color:var(--color-surface-elevated)] hover:text-[color:var(--color-accent)]"
+          onKeyDown={event => {
+            event.stopPropagation();
+          }}
           onClick={event => {
             event.stopPropagation();
             onToggleStar?.(item.id);
           }}
           type="button"
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
    <div
      className={cn(
        "flex w-full items-start gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors",
        active
          ? "border-[color:var(--color-accent-dim)] bg-[color:var(--color-accent-tint)]"
          : "border-transparent hover:border-[color:var(--color-divider)] hover:bg-[color:var(--color-surface)]"
      )}
      data-testid={`network-room-${item.roomType}-${item.id}`}
      onKeyDown={event => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onSelect(item);
        }
      }}
      onClick={() => onSelect(item)}
      role="button"
      tabIndex={0}
    >
      <div className="mt-0.5 flex items-center gap-2">
        {isChannel ? (
          <Hash className="size-3.5 text-[color:var(--color-text-secondary)]" />
        ) : (
          <StatusDot tone={item.tone} />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span
            className={cn(
              "truncate text-[13px] font-medium",
              item.unreadCount > 0
                ? "text-[color:var(--color-text-primary)]"
                : "text-[color:var(--color-text-secondary)]"
            )}
          >
            {item.title}
          </span>
          {item.unreadCount > 0 ? (
            <MonoBadge className="ml-auto" tone="accent">
              {item.unreadCount}
            </MonoBadge>
          ) : null}
        </div>
        <div className="mt-1 flex items-center gap-2">
          <span className="truncate text-[12px] text-[color:var(--color-text-tertiary)]">
            {item.preview}
          </span>
        </div>
        <div className="mt-1 flex items-center gap-2">
          <span className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
            {item.meta}
          </span>
          <span className="text-[color:var(--color-text-tertiary)]">·</span>
          <span className="truncate font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
            {item.subtitle}
          </span>
        </div>
      </div>
      {isChannel ? (
        <button
          aria-label={item.isStarred ? "Unstar channel" : "Star channel"}
          className="mt-0.5 rounded-md p-1 text-[color:var(--color-text-tertiary)] transition-colors hover:bg-[color:var(--color-surface-elevated)] hover:text-[color:var(--color-accent)]"
          onKeyDown={event => {
            event.stopPropagation();
          }}
          onClick={event => {
            event.stopPropagation();
            onToggleStar?.(item.id);
          }}
          type="button"
        >
          <Sparkles
            className={cn("size-3.5", item.isStarred && "text-[color:var(--color-accent)]")}
          />
        </button>
      ) : null}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/network/components/network-workspace-shell.tsx` around lines
174 - 246, The row's onKeyDown handler (the anonymous handler that checks
event.key === "Enter" || " ") currently catches keyboard events for the whole
item; add an onKeyDown on the star button (the button that renders Sparkles and
calls onToggleStar) that stops propagation so keyboard presses on the star don't
bubble to the row. Concretely, in the star button element add an onKeyDown
handler that calls event.stopPropagation() (and for safety you can check keys
"Enter" and " " before stopping) so keyboard activation still triggers the
button but won't trigger the row's onSelect.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Reasoning: the current `NetworkSidebarRow` implementation no longer attaches keyboard selection handling to a shared row container. Room selection happens on the main room button, while the star control is a sibling button, so keyboard activation on the star button does not bubble into a row-level `onSelect` handler in the current code.
- Resolution: no code change. The current room button and sibling star button structure already prevents the reported keyboard-bubbling path.
- Verification: `bun run test:raw src/routes/_app/-network.test.tsx src/systems/network/components/network-create-channel-dialog.test.tsx`, `make web-lint`, `make web-typecheck`, and `make verify`
