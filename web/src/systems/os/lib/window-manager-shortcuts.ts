import {
  WINDOW_MANAGER_ACTIONS,
  type WindowManagerActionDefinition,
} from "./window-manager-command-registry";

type ShortcutModifier = "meta" | "control" | "alt" | "shift";

export interface ParsedShortcutChord {
  modifiers: ReadonlySet<ShortcutModifier>;
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
    rawModifiers.some(modifier => !MODIFIER_ORDER.includes(modifier as ShortcutModifier))
  ) {
    return null;
  }
  const modifiers = new Set(rawModifiers as ShortcutModifier[]);
  const ordered = MODIFIER_ORDER.filter(modifier => modifiers.has(modifier));
  return { modifiers, code, canonical: [...ordered, code].join("+") };
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
  return [...labels, codeLabel].join("");
}

export function resolveWindowManagerActions(
  overrides: Readonly<Record<string, string>>
): readonly ResolvedWindowManagerAction[] {
  return WINDOW_MANAGER_ACTIONS.map(action => {
    const raw = overrides[action.id] ?? action.defaultChord;
    const chord = raw ? parseShortcutChord(raw) : null;
    return { ...action, chord, shortcutLabel: chord ? shortcutLabel(chord) : null };
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
