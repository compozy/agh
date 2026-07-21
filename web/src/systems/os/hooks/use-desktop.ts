import { useStore } from "zustand";

import { useOsShell } from "./use-os-shell";
import type { OsDesktopRuntimeStore } from "../lib/os-types";

/** Selector-scoped read of the shell's desktop store (runtime contract). */
export function useDesktop<T>(selector: (state: OsDesktopRuntimeStore) => T): T {
  const { store } = useOsShell();
  return useStore(store, selector);
}
