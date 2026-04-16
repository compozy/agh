"use client";

import { useEffect, useState } from "react";

/**
 * Returns true when the user has requested reduced motion via the OS.
 * SSR-safe: defaults to false server-side and on first client render.
 */
export function useReducedMotion(): boolean {
  const [reduced, setReduced] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return;
    }
    const query = window.matchMedia("(prefers-reduced-motion: reduce)");
    setReduced(query.matches);
    const handler = (event: MediaQueryListEvent) => setReduced(event.matches);
    query.addEventListener("change", handler);
    return () => query.removeEventListener("change", handler);
  }, []);

  return reduced;
}
