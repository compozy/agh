"use client";

import * as React from "react";

export interface MeasuredBounds {
  width: number;
  height: number;
}

export type UseMeasureResult<T extends HTMLElement = HTMLElement> = readonly [
  (node: T | null) => void,
  MeasuredBounds,
];

export function useMeasure<T extends HTMLElement = HTMLElement>(): UseMeasureResult<T> {
  const [bounds, setBounds] = React.useState<MeasuredBounds>({ width: 0, height: 0 });
  const observerRef = React.useRef<ResizeObserver | null>(null);

  const refCallback = React.useCallback((node: T | null) => {
    observerRef.current?.disconnect();
    if (!node || typeof ResizeObserver === "undefined") {
      observerRef.current = null;
      return;
    }
    const observer = new ResizeObserver(entries => {
      const entry = entries[0];
      if (!entry) return;
      const { width, height } = entry.contentRect;
      setBounds(prev =>
        prev.width === width && prev.height === height ? prev : { width, height }
      );
    });
    observer.observe(node);
    observerRef.current = observer;
  }, []);

  React.useEffect(() => () => observerRef.current?.disconnect(), []);

  return [refCallback, bounds] as const;
}
