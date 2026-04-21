# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Author 14 story files under `web/src/components/ui/stories/` for the remaining shadcn-layer composites (field, input-group, button-group, select, native-select, textarea, toggle, toggle-group, switch, avatar, empty, item, direction, sonner).
- Verify via web lint, typecheck, and `build-storybook` (no Vitest per shared workflow decision inherited from task_02/task_03).

## Important Decisions

- Avatar "image" story uses an inline `data:image/svg+xml` URI — no cross-repo asset existed and data URIs keep the story offline-safe under Storybook's MSW bypass.
- Fallback avatar story uses `https://example.invalid/...` so the image deterministically fails and the AvatarFallback initials render.
- Sonner stories mount the `Toaster` inside each `render()` (no provider-level mount) per task spec to prevent cross-story leakage.
- Direction stories wrap the paragraph in both `DirectionProvider` (context) and a native `<div dir="...">` so the DOM `dir` attribute is asserted by the task spec property test.
- Field error stories wire `aria-invalid` + `aria-describedby` to a matching `FieldError id` so assistive tech associates the error with the input.

## Learnings

- `ToggleGroup` multi-select requires both `multiple` and an array `defaultValue`/`value`; `Toggle` items expose `value: Value | undefined` for the group to track.
- `InputGroup` expects addons with `align="block-end"` to stack vertical controls (e.g., textarea + send button); default `align` is `"inline-start"`.
- `FieldLegend` is used directly under `FieldSet` — not inside `Field`. Group field rows under `FieldGroup` to get the responsive inline layout.
- Sonner's generated toasts use `role="status"` (not `role="alert"`) for non-critical variants; the task spec test expects exactly `"status"`.
- oxfmt will rewrite single-line JSX children (e.g., `<Kbd><CommandIcon />K</Kbd>`) and normalize string quotes — author close to the final shape to minimize diffs.

## Files / Surfaces

- New: `web/src/components/ui/stories/{field,input-group,button-group,select,native-select,textarea,toggle,toggle-group,switch,avatar,empty,item,direction,sonner}.stories.tsx` (14 files).
- No changes to components, `web/.storybook/preview.ts`, or MSW handlers — infrastructure from task_01/task_03 was sufficient.

## Errors / Corrections

- First `item` story used a bare `<>` inside `.map()`, which would have triggered missing-key warnings; switched to `Fragment key={agent.id}` before first verification.

## Ready for Next Run

- Task_05 can proceed as planned; the form primitives now have a visual reference surface that system stories may import via `@agh/ui` / `@/components/ui` paths.
- No open risks uncovered during this task.
