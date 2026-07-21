import type { OsSnapZoneId } from "./os-types";

/**
 * The keyboard/palette snap surface (ADR-009): one row per zone + restore,
 * shared by the palette commands, the ⌃⌥ chord handler, and the Help menu so
 * the three listings can never drift. The palette is the guaranteed keyboard
 * path where a browser reserves a chord.
 */
export interface OsSnapCommand {
  /** Target zone; `null` restores the pre-snap rect exactly. */
  zoneId: OsSnapZoneId | null;
  label: string;
  /** Display chord (Help menu, palette shortcut hint). */
  keys: string;
  /** `KeyboardEvent.code` the ⌃⌥ chord matches (layout-independent). */
  code: string;
}

export const OS_SNAP_COMMANDS: readonly OsSnapCommand[] = [
  { zoneId: "left", label: "Snap left half", keys: "⌃⌥←", code: "ArrowLeft" },
  { zoneId: "right", label: "Snap right half", keys: "⌃⌥→", code: "ArrowRight" },
  { zoneId: "top-left", label: "Snap top left quarter", keys: "⌃⌥U", code: "KeyU" },
  { zoneId: "top-right", label: "Snap top right quarter", keys: "⌃⌥I", code: "KeyI" },
  { zoneId: "bottom-left", label: "Snap bottom left quarter", keys: "⌃⌥J", code: "KeyJ" },
  { zoneId: "bottom-right", label: "Snap bottom right quarter", keys: "⌃⌥K", code: "KeyK" },
  { zoneId: null, label: "Restore window size", keys: "⌃⌥↓", code: "ArrowDown" },
];
