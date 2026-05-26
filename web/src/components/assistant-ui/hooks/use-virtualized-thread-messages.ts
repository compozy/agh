import { useVirtualizer } from "@tanstack/react-virtual";
import { type RefObject, useCallback, useEffect, useRef } from "react";

export function useVirtualizedThreadMessages(
  viewportRef: RefObject<HTMLDivElement | null>,
  messageCount: number
) {
  const shouldStickToBottomRef = useRef(true);
  const virtualizer = useVirtualizer({
    count: messageCount,
    getScrollElement: () => viewportRef.current,
    estimateSize: () => 144,
    overscan: 8,
  });

  const updateStickiness = useCallback(() => {
    const viewport = viewportRef.current;
    if (!viewport) {
      shouldStickToBottomRef.current = true;
      return;
    }
    const distanceFromBottom = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
    shouldStickToBottomRef.current = distanceFromBottom < 96;
  }, [viewportRef]);

  useEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport) {
      return undefined;
    }
    viewport.addEventListener("scroll", updateStickiness, { passive: true });
    updateStickiness();
    return () => {
      viewport.removeEventListener("scroll", updateStickiness);
    };
  }, [updateStickiness, viewportRef]);

  useEffect(() => {
    if (messageCount === 0 || !shouldStickToBottomRef.current) {
      return;
    }
    virtualizer.scrollToIndex(messageCount - 1, { align: "end" });
  }, [messageCount, virtualizer]);

  return { virtualizer };
}
