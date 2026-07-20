import { useStore } from "zustand";

import { useOsShell } from "./use-os-shell";
import type { OsDesktopStore } from "../lib/os-types";

/** Selector-scoped read of the shell's desktop store. */
export function useDesktop<T>(selector: (state: OsDesktopStore) => T): T {
  const { store } = useOsShell();
  return useStore(store, selector);
}
