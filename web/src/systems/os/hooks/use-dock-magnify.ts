import { useEffect, useRef, useSyncExternalStore, type RefObject } from "react";

import { DOCK_MAG_RADIUS, dockMagnifyFactor, dockMagnifyTransform } from "../lib/dock-magnify";

const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)";
const DOCK_ITEM_SELECTOR = '[data-slot="os-dock-item"]';

function subscribeReducedMotion(callback: () => void): () => void {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return () => undefined;
  }
  const mql = window.matchMedia(REDUCED_MOTION_QUERY);
  if (typeof mql.addEventListener === "function") {
    mql.addEventListener("change", callback);
    return () => mql.removeEventListener("change", callback);
  }
  mql.addListener(callback);
  return () => mql.removeListener(callback);
}

function getReducedMotion(): boolean {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return false;
  }
  return window.matchMedia(REDUCED_MOTION_QUERY).matches;
}

function clearTransforms(root: HTMLElement): void {
  for (const item of root.querySelectorAll<HTMLElement>(DOCK_ITEM_SELECTOR)) {
    item.style.transform = "";
  }
}

function applyMagnify(root: HTMLElement, clientX: number): void {
  for (const item of root.querySelectorAll<HTMLElement>(DOCK_ITEM_SELECTOR)) {
    const rect = item.getBoundingClientRect();
    const distance = Math.abs(clientX - (rect.left + rect.width / 2));
    const factor = dockMagnifyFactor(distance, DOCK_MAG_RADIUS);
    item.style.transform = dockMagnifyTransform(factor);
  }
}

/**
 * OpenDesign dock magnification: pointer proximity scales/lifts every dock
 * item inside `rootRef` with a linear falloff. Disabled when `enabled` is
 * false (compact presentation) or the user prefers reduced motion.
 */
export function useDockMagnify(rootRef: RefObject<HTMLElement | null>, enabled = true): void {
  const reduceMotion = useSyncExternalStore(subscribeReducedMotion, getReducedMotion, () => false);
  const active = enabled && !reduceMotion;
  const pendingX = useRef<number | null>(null);
  const rafId = useRef(0);

  useEffect(() => {
    const root = rootRef.current;
    if (!root || !active) {
      if (root) clearTransforms(root);
      return;
    }

    const flush = () => {
      rafId.current = 0;
      const x = pendingX.current;
      if (x === null) return;
      applyMagnify(root, x);
    };

    const onMove = (event: PointerEvent) => {
      pendingX.current = event.clientX;
      if (rafId.current === 0) {
        rafId.current = requestAnimationFrame(flush);
      }
    };

    const onLeave = () => {
      pendingX.current = null;
      if (rafId.current !== 0) {
        cancelAnimationFrame(rafId.current);
        rafId.current = 0;
      }
      clearTransforms(root);
    };

    root.addEventListener("pointermove", onMove);
    root.addEventListener("pointerleave", onLeave);
    return () => {
      root.removeEventListener("pointermove", onMove);
      root.removeEventListener("pointerleave", onLeave);
      if (rafId.current !== 0) {
        cancelAnimationFrame(rafId.current);
        rafId.current = 0;
      }
      clearTransforms(root);
    };
  }, [rootRef, active]);
}
