import { useVirtualizer, type Virtualizer } from "@tanstack/react-virtual";
import { type RefObject, useCallback, useEffect, useRef, useState } from "react";

import {
  createVirtualizerMeasurementState,
  getAnchoredTurnMetrics,
  nextScrollMode,
  type TimelineScrollMode,
} from "../timeline-scroll-anchoring";
import { estimateMessageSize } from "../timeline-row-estimates";

// Distance (px) from the live edge within which the viewport counts as "at the
// bottom": inside it, live-follow resumes and the pill hides. Mirrors T3's isAtEnd.
const AT_END_THRESHOLD = 96;
// Gap (px) kept above a freshly-anchored prompt so it sits just below the top edge
// while its answer streams in below it.
const ANCHOR_OFFSET = 16;
// Our composer is a flex sibling below the scroll viewport (not an overlay), so the
// viewport height already excludes it — no overlay compensation is needed.
const COMPOSER_OVERLAY_HEIGHT = 0;

interface ThreadScrollMessage {
  id?: string;
  role?: string;
}

export interface VirtualizedThreadMessagesResult {
  virtualizer: Virtualizer<HTMLDivElement, Element>;
  showScrollToBottom: boolean;
  scrollToEnd: () => void;
}

function findLastUserIndex(messages: readonly ThreadScrollMessage[]): number {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    if (messages[index]?.role === "user") {
      return index;
    }
  }
  return -1;
}

/**
 * Drives the transcript virtualizer with an explicit three-mode live-follow
 * machine ported from T3 (`timeline-scroll-anchoring.ts`):
 *
 * - `following-end` pins the viewport to the newest message as content streams;
 * - a manual wheel/touch/pointer gesture flips to `free-scrolling` and reveals the
 *   scroll-to-bottom pill; reaching the live edge (or clicking the pill) resumes;
 * - a freshly-sent prompt flips to `anchoring-new-turn`, pinning the user message
 *   near the top and revealing its streaming answer below it.
 */
export function useVirtualizedThreadMessages(
  viewportRef: RefObject<HTMLDivElement | null>,
  messages: readonly ThreadScrollMessage[]
): VirtualizedThreadMessagesResult {
  const messageCount = messages.length;
  const modeRef = useRef<TimelineScrollMode>("following-end");
  const anchorIndexRef = useRef<number | null>(null);
  const lastUserIdRef = useRef<string | null>(null);
  const previousCountRef = useRef(0);
  const [showScrollToBottom, setShowScrollToBottom] = useState(false);

  const estimateSize = useCallback(
    (index: number) => estimateMessageSize(messages[index]),
    [messages]
  );

  // Key the virtualizer's measurement cache by message identity (T3's
  // `keyExtractor`): a settled message's measured height and React key follow the
  // message across appends/reconnects instead of being pinned to a shifting index,
  // so recycled rows recycle by the row they actually are.
  const getItemKey = useCallback((index: number) => messages[index]?.id ?? index, [messages]);

  const virtualizer = useVirtualizer({
    count: messageCount,
    getScrollElement: () => viewportRef.current,
    estimateSize,
    getItemKey,
    overscan: 8,
  });

  const distanceFromBottom = useCallback(() => {
    const viewport = viewportRef.current;
    if (!viewport) {
      return 0;
    }
    return viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
  }, [viewportRef]);

  const scrollToEnd = useCallback(() => {
    modeRef.current = nextScrollMode(modeRef.current, "scroll-to-end");
    anchorIndexRef.current = null;
    setShowScrollToBottom(false);
    if (messageCount > 0) {
      virtualizer.scrollToIndex(messageCount - 1, { align: "end" });
    }
  }, [messageCount, virtualizer]);

  // Manual navigation opts out of live-follow; the scroll listener resumes it at
  // the live edge and toggles the pill while free-scrolling.
  useEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport) {
      return undefined;
    }

    const handleManualNavigation = () => {
      modeRef.current = nextScrollMode(modeRef.current, "manual-navigation");
      anchorIndexRef.current = null;
      setShowScrollToBottom(distanceFromBottom() > AT_END_THRESHOLD);
    };
    const handleScroll = () => {
      if (distanceFromBottom() <= AT_END_THRESHOLD) {
        modeRef.current = nextScrollMode(modeRef.current, "reached-end");
        setShowScrollToBottom(false);
        return;
      }
      if (modeRef.current === "free-scrolling") {
        setShowScrollToBottom(true);
      }
    };

    viewport.addEventListener("wheel", handleManualNavigation, { passive: true });
    viewport.addEventListener("touchmove", handleManualNavigation, { passive: true });
    viewport.addEventListener("pointerdown", handleManualNavigation, { passive: true });
    viewport.addEventListener("scroll", handleScroll, { passive: true });
    return () => {
      viewport.removeEventListener("wheel", handleManualNavigation);
      viewport.removeEventListener("touchmove", handleManualNavigation);
      viewport.removeEventListener("pointerdown", handleManualNavigation);
      viewport.removeEventListener("scroll", handleScroll);
    };
  }, [viewportRef, distanceFromBottom]);

  // New content settles the mode: a fresh prompt anchors, and the follow/anchor
  // effect repositions the viewport for the current mode.
  useEffect(() => {
    const lastUserIndex = findLastUserIndex(messages);
    const lastUserId = lastUserIndex >= 0 ? (messages[lastUserIndex]?.id ?? null) : null;
    const grew = messageCount > previousCountRef.current;
    // Only a prompt sent into an already-open thread anchors; the initial
    // population is a cold open that should land at the live edge.
    if (
      previousCountRef.current > 0 &&
      grew &&
      lastUserId &&
      lastUserId !== lastUserIdRef.current
    ) {
      modeRef.current = nextScrollMode(modeRef.current, "new-turn");
      anchorIndexRef.current = lastUserIndex;
      setShowScrollToBottom(false);
    }
    lastUserIdRef.current = lastUserId;
    previousCountRef.current = messageCount;

    if (messageCount === 0) {
      return;
    }

    if (modeRef.current === "anchoring-new-turn" && anchorIndexRef.current !== null) {
      const state = createVirtualizerMeasurementState(
        virtualizer,
        viewportRef.current?.clientHeight ?? 0
      );
      const metrics = getAnchoredTurnMetrics({
        state,
        anchorIndex: anchorIndexRef.current,
        composerOverlayHeight: COMPOSER_OVERLAY_HEIGHT,
        anchorOffset: ANCHOR_OFFSET,
      });
      if (!metrics) {
        return;
      }
      if (metrics.overflowsUsableViewport) {
        // The turn is taller than the viewport: reveal the newest output while
        // keeping the anchored prompt as high as possible.
        if (metrics.scrollDeltaToRevealEnd > 1) {
          virtualizer.scrollToOffset(state.scroll + metrics.scrollDeltaToRevealEnd);
        }
      } else {
        // The turn fits: pin the prompt just below the top edge.
        virtualizer.scrollToOffset(Math.max(0, metrics.anchorTop - ANCHOR_OFFSET));
      }
      return;
    }

    if (modeRef.current === "following-end") {
      virtualizer.scrollToIndex(messageCount - 1, { align: "end" });
    }
  }, [messages, messageCount, virtualizer, viewportRef]);

  return { virtualizer, showScrollToBottom, scrollToEnd };
}
