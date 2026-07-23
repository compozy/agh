import type { OsArrangePreset, OsDesktopRuntimeStore, WindowManagerController } from "./os-types";
import type { SnapCorner, SnapSide } from "./snap-targets";

export type WindowPlacementId = SnapSide | SnapCorner;
export type WindowManagerActionId =
  | `window.tile.${WindowPlacementId}`
  | "window.close"
  | "window.minimize"
  | "window.zoom"
  | "window.toggle_floating"
  | "window.focus.left"
  | "window.focus.right"
  | "window.focus.up"
  | "window.focus.down"
  | "desktop.switch.previous"
  | "desktop.switch.next"
  | "desktop.overview"
  | "layout.arrange.two-up"
  | "layout.arrange.grid"
  | "layout.balance"
  | "layout.undo"
  | "layout.redo";

export interface WindowPlacementCommand {
  id: `window.tile.${WindowPlacementId}`;
  placement: WindowPlacementId;
  label: string;
}

export interface WindowArrangeCommand {
  id: `layout.arrange.${OsArrangePreset}`;
  preset: OsArrangePreset;
  label: string;
}

export interface WindowManagerActionDefinition {
  id: WindowManagerActionId;
  label: string;
  defaultChord?: string;
  needsFocusedWindow?: boolean;
}

export interface ParsedShortcutChord {
  modifiers: ReadonlySet<"meta" | "control" | "alt" | "shift">;
  code: string;
  canonical: string;
}

export interface ResolvedWindowManagerAction extends WindowManagerActionDefinition {
  chord: ParsedShortcutChord | null;
  shortcutLabel: string | null;
}

const MODIFIER_ORDER = ["meta", "control", "alt", "shift"] as const;
const KEY_CODE_PATTERN =
  /^(?:Key[A-Z]|Digit[0-9]|Arrow(?:Left|Right|Up|Down)|Bracket(?:Left|Right)|Comma|Period|Slash|Semicolon|Quote|Backquote|Minus|Equal|Backslash|Enter|Space|Tab|Escape|Backspace|Delete|Home|End|PageUp|PageDown|F(?:[1-9]|1[0-2]))$/;

export const WINDOW_PLACEMENT_COMMANDS: readonly WindowPlacementCommand[] = [
  { id: "window.tile.left", placement: "left", label: "Tile left half" },
  { id: "window.tile.right", placement: "right", label: "Tile right half" },
  { id: "window.tile.top", placement: "top", label: "Tile top half" },
  { id: "window.tile.bottom", placement: "bottom", label: "Tile bottom half" },
  {
    id: "window.tile.top-left",
    placement: "top-left",
    label: "Tile top left quarter",
  },
  {
    id: "window.tile.top-right",
    placement: "top-right",
    label: "Tile top right quarter",
  },
  {
    id: "window.tile.bottom-left",
    placement: "bottom-left",
    label: "Tile bottom left quarter",
  },
  {
    id: "window.tile.bottom-right",
    placement: "bottom-right",
    label: "Tile bottom right quarter",
  },
];

export const WINDOW_ARRANGE_COMMANDS: readonly WindowArrangeCommand[] = [
  {
    id: "layout.arrange.two-up",
    preset: "two-up",
    label: "Arrange left & right",
  },
  { id: "layout.arrange.grid", preset: "grid", label: "Arrange in grid" },
];

export const WINDOW_MANAGER_ACTIONS: readonly WindowManagerActionDefinition[] = [
  {
    id: "window.close",
    label: "Close window",
    defaultChord: "meta+KeyW",
    needsFocusedWindow: true,
  },
  {
    id: "window.minimize",
    label: "Minimize window",
    defaultChord: "meta+KeyM",
    needsFocusedWindow: true,
  },
  {
    id: "window.zoom",
    label: "Zoom window",
    defaultChord: "control+alt+ArrowUp",
    needsFocusedWindow: true,
  },
  {
    id: "window.toggle_floating",
    label: "Toggle floating",
    defaultChord: "control+alt+KeyF",
    needsFocusedWindow: true,
  },
  ...WINDOW_PLACEMENT_COMMANDS.map(command => ({
    ...command,
    defaultChord:
      command.placement === "left"
        ? "control+alt+ArrowLeft"
        : command.placement === "right"
          ? "control+alt+ArrowRight"
          : command.placement === "top-left"
            ? "control+alt+KeyU"
            : command.placement === "top-right"
              ? "control+alt+KeyI"
              : command.placement === "bottom-left"
                ? "control+alt+KeyJ"
                : command.placement === "bottom-right"
                  ? "control+alt+KeyK"
                  : undefined,
    needsFocusedWindow: true,
  })),
  {
    id: "window.focus.left",
    label: "Focus left",
    defaultChord: "control+ArrowLeft",
  },
  {
    id: "window.focus.right",
    label: "Focus right",
    defaultChord: "control+ArrowRight",
  },
  { id: "window.focus.up", label: "Focus up", defaultChord: "control+ArrowUp" },
  {
    id: "window.focus.down",
    label: "Focus down",
    defaultChord: "control+ArrowDown",
  },
  {
    id: "desktop.switch.previous",
    label: "Previous desktop",
    defaultChord: "control+alt+BracketLeft",
  },
  {
    id: "desktop.switch.next",
    label: "Next desktop",
    defaultChord: "control+alt+BracketRight",
  },
  {
    id: "desktop.overview",
    label: "Desktops overview",
    defaultChord: "meta+shift+KeyS",
  },
  ...WINDOW_ARRANGE_COMMANDS.map(command => ({
    ...command,
    needsFocusedWindow: true,
  })),
  {
    id: "layout.balance",
    label: "Balance layout",
    defaultChord: "control+alt+KeyB",
    needsFocusedWindow: true,
  },
  { id: "layout.undo", label: "Undo layout", defaultChord: "meta+KeyZ" },
  {
    id: "layout.redo",
    label: "Redo layout",
    defaultChord: "meta+shift+KeyZ",
  },
];

export const WINDOW_MANAGER_ACTION_IDS = new Set(WINDOW_MANAGER_ACTIONS.map(action => action.id));

export function isWindowManagerActionId(value: string): value is WindowManagerActionId {
  return WINDOW_MANAGER_ACTION_IDS.has(value as WindowManagerActionId);
}

export function parseShortcutChord(value: string): ParsedShortcutChord | null {
  const tokens = value
    .split("+")
    .map(token => token.trim())
    .filter(Boolean);
  if (tokens.length < 2) return null;
  const code = tokens.at(-1) ?? "";
  if (!KEY_CODE_PATTERN.test(code)) return null;
  const rawModifiers = tokens.slice(0, -1).map(token => token.toLowerCase());
  if (
    new Set(rawModifiers).size !== rawModifiers.length ||
    rawModifiers.some(
      modifier => !MODIFIER_ORDER.includes(modifier as (typeof MODIFIER_ORDER)[number])
    )
  ) {
    return null;
  }
  const modifiers = new Set(rawModifiers as Array<(typeof MODIFIER_ORDER)[number]>);
  const ordered = MODIFIER_ORDER.filter(modifier => modifiers.has(modifier));
  return {
    modifiers,
    code,
    canonical: [...ordered, code].join("+"),
  };
}

export function shortcutLabel(chord: ParsedShortcutChord): string {
  const modifierLabels = { meta: "⌘", control: "⌃", alt: "⌥", shift: "⇧" };
  const codeLabel = chord.code
    .replace(/^Key/, "")
    .replace(/^Digit/, "")
    .replace("ArrowLeft", "←")
    .replace("ArrowRight", "→")
    .replace("ArrowUp", "↑")
    .replace("ArrowDown", "↓")
    .replace("BracketLeft", "[")
    .replace("BracketRight", "]");
  const labels: string[] = [];
  for (const modifier of MODIFIER_ORDER) {
    if (chord.modifiers.has(modifier)) labels.push(modifierLabels[modifier]);
  }
  labels.push(codeLabel);
  return labels.join("");
}

export function resolveWindowManagerActions(
  overrides: Readonly<Record<string, string>>
): readonly ResolvedWindowManagerAction[] {
  return WINDOW_MANAGER_ACTIONS.map(action => {
    const raw = overrides[action.id] ?? action.defaultChord;
    const chord = raw ? parseShortcutChord(raw) : null;
    return {
      ...action,
      chord,
      shortcutLabel: chord ? shortcutLabel(chord) : null,
    };
  });
}

export function shortcutMatches(event: KeyboardEvent, chord: ParsedShortcutChord): boolean {
  return (
    event.code === chord.code &&
    event.metaKey === chord.modifiers.has("meta") &&
    event.ctrlKey === chord.modifiers.has("control") &&
    event.altKey === chord.modifiers.has("alt") &&
    event.shiftKey === chord.modifiers.has("shift")
  );
}

export function dispatchWindowPlacement(
  manager: WindowManagerController,
  windowId: string,
  command: WindowPlacementCommand
): void {
  manager.tileWindow(windowId, command.placement);
}

export function dispatchWindowManagerAction(
  actionId: WindowManagerActionId,
  context: {
    manager: WindowManagerController;
    state: OsDesktopRuntimeStore;
    openDesktops: () => void;
  }
): void {
  const { manager, state } = context;
  const focusedId = state.focusedId;
  if (actionId === "desktop.overview") return context.openDesktops();
  if (actionId === "desktop.switch.previous") return manager.switchDesktopDirection("previous");
  if (actionId === "desktop.switch.next") return manager.switchDesktopDirection("next");
  if (actionId === "layout.undo") return manager.undoLayout();
  if (actionId === "layout.redo") return manager.redoLayout();
  if (actionId.startsWith("window.focus.")) {
    return manager.focusDirection(
      actionId.slice("window.focus.".length) as "left" | "right" | "up" | "down"
    );
  }
  if (focusedId === null) return;
  if (actionId === "window.close") {
    void state.closeWindow(focusedId);
    return;
  }
  if (actionId === "window.minimize") {
    void state.minimizeWindow(focusedId);
    return;
  }
  if (actionId === "window.zoom") return state.zoomWindow(focusedId);
  if (actionId === "window.toggle_floating") return state.toggleFloating(focusedId);
  if (actionId === "layout.balance") return manager.balanceFocusedLayout();
  const placement = WINDOW_PLACEMENT_COMMANDS.find(command => command.id === actionId);
  if (placement && state.windowManagerConfig) {
    return dispatchWindowPlacement(manager, focusedId, placement);
  }
  const arrangement = WINDOW_ARRANGE_COMMANDS.find(command => command.id === actionId);
  if (arrangement) state.arrangeLayout(focusedId, arrangement.preset);
}
