/**
 * Single global owner for the `⌘K` / `Ctrl+K` runtime-selector shortcut.
 *
 * Every mounted `RuntimeSelector` registers here, but there is exactly ONE
 * `document` keydown listener for all of them (installed on first register,
 * removed on last unregister). Per-instance document listeners are forbidden:
 * multiple mounted selectors would each install a competing handler and fight
 * over one event. This registry resolves a single deterministic owner instead.
 *
 * Ownership rule (explicit, TOTAL, in order):
 *  1. Among eligible + visible + connected selectors, an open popup owns `⌘K`.
 *  2. Otherwise, the last-focused eligible selector owns it.
 *     one owns it.
 *  3. Otherwise the most-recently-registered eligible selector owns it — a
 *     deterministic fallback so several eligible selectors with no focus never
 *     degrade to "no owner". This also covers the common single-selector page
 *     (`⌘K` works without first focusing the trigger).
 *
 * Hidden or unmounted selectors never own the shortcut: unmounting unregisters
 * the entry, and a mounted-but-hidden trigger fails the visibility check.
 */
export interface CommandKEntry {
  id: string;
  /** The selector is interactive (not disabled). */
  isEligible: () => boolean;
  /** The trigger element is connected and visible in the layout. */
  isVisible: () => boolean;
  /** The selector's popup is currently open. */
  isOpen: () => boolean;
  open: () => void;
  close: () => void;
}

const entries = new Map<string, CommandKEntry>();
let lastFocusedId: string | null = null;
let listening = false;

function isCommandK(event: KeyboardEvent): boolean {
  return (
    (event.metaKey || event.ctrlKey) &&
    !event.altKey &&
    !event.shiftKey &&
    event.key.toLowerCase() === "k"
  );
}

function resolveOwner(): CommandKEntry | null {
  const candidates = [...entries.values()].filter(entry => entry.isEligible() && entry.isVisible());
  // Rule 1: an eligible, visible open selector owns the shortcut.
  for (const entry of candidates) {
    if (entry.isOpen()) return entry;
  }
  if (candidates.length === 0) return null;
  // Rule 2: the most-recently-focused eligible selector wins.
  if (lastFocusedId) {
    const focused = candidates.find(entry => entry.id === lastFocusedId);
    if (focused) return focused;
  }
  // Rule 3: deterministic fallback — the most-recently-registered eligible
  // selector (last in insertion order). Ownership stays total: multiple eligible
  // selectors with no focus never resolve to "no owner".
  return candidates[candidates.length - 1];
}

function handleKeyDown(event: KeyboardEvent): void {
  if (!isCommandK(event)) return;
  const owner = resolveOwner();
  if (!owner) return;
  event.preventDefault();
  if (owner.isOpen()) owner.close();
  else owner.open();
}

export function registerCommandK(entry: CommandKEntry): () => void {
  entries.set(entry.id, entry);
  if (!listening && typeof document !== "undefined") {
    document.addEventListener("keydown", handleKeyDown);
    listening = true;
  }
  return () => {
    entries.delete(entry.id);
    if (lastFocusedId === entry.id) lastFocusedId = null;
    if (entries.size === 0 && listening && typeof document !== "undefined") {
      document.removeEventListener("keydown", handleKeyDown);
      listening = false;
    }
  };
}

/** Mark a selector as the last-focused owner candidate for `⌘K`. */
export function noteCommandKFocus(id: string): void {
  lastFocusedId = id;
}

export function triggerVisible(element: HTMLElement | null): boolean {
  if (!element || !element.isConnected) return false;
  // aria-hidden ancestors are a11y-hidden even when layout-visible.
  // checkVisibility() does not cover this, so gate it before the platform check.
  if (element.closest('[aria-hidden="true"]')) return false;
  // Real browsers: the platform check honors display/visibility/content-visibility
  // and the `hidden` attribute. jsdom has no layout engine, so fall back to a
  // connected-and-not-in-a-hidden-subtree heuristic.
  if (typeof element.checkVisibility === "function") return element.checkVisibility();
  return !element.closest("[hidden]");
}
