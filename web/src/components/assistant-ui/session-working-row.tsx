import { useEffect, useRef } from "react";

import { TypingDots } from "@agh/ui";
import { usePrefersReducedMotion } from "./hooks/use-prefers-reduced-motion";
import type { SessionWorkingRow } from "./session-timeline.logic";

// Live "Working for Xs" formatting mirrors T3Code/Synara `formatWorkingTimer`:
// floor to whole seconds from the turn start (0s on the first tick), roll up to
// `Xm Ys` past a minute and `Xh Ym` past an hour. Distinct from the settled
// fold's `formatDuration` (which rounds to a minimum of 1s) because the live
// timer counts up from 0 while the turn is still in flight.
export function formatWorkingElapsed(startedAtMs: number, nowMs: number): string {
  const totalSeconds = Math.max(0, Math.floor((nowMs - startedAtMs) / 1000));
  if (totalSeconds < 60) {
    return `${totalSeconds}s`;
  }
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
  }
  return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
}

// Self-ticking elapsed label. Mirrors T3Code's `WorkingTimer` (and Synara's twin):
// the span mutates its own text node on a one-second interval, so a streaming turn
// never forces a React commit per second — the whole transcript tree stays put
// while the clock advances (proven by the render-count probe in the thread suite).
function WorkingTimer({ startedAt }: { startedAt: number }) {
  const textRef = useRef<HTMLSpanElement>(null);
  const initialText = formatWorkingElapsed(startedAt, Date.now());

  useEffect(() => {
    const update = () => {
      if (textRef.current) {
        textRef.current.textContent = formatWorkingElapsed(startedAt, Date.now());
      }
    };
    update();
    const id = window.setInterval(update, 1000);
    return () => window.clearInterval(id);
  }, [startedAt]);

  return (
    <span ref={textRef} data-testid="session-working-timer" className="tabular-nums">
      {initialText}
    </span>
  );
}

/**
 * Presentational streaming indicator, split from the row wrapper so Storybook can
 * render both the motion and reduced-motion variants without touching `matchMedia`.
 *
 * Motion: T3Code-style typing dots (`@agh/ui` `TypingDots`, wiring the
 * `typing-bounce` keyframe on `bg-subtle` dots) beside a live "Working for Xs"
 * `tabular-nums` timer. Reduced motion degrades to a resting label — no
 * `TypingDots`, no ticking timer — so no animation class reaches the DOM.
 */
export function WorkingIndicator({
  startedAt,
  reducedMotion,
}: {
  startedAt?: number;
  reducedMotion: boolean;
}) {
  if (reducedMotion) {
    return (
      <div
        role="status"
        aria-label="Working"
        data-testid="session-working-row"
        className="flex items-center gap-2 text-small-body text-subtle"
      >
        <span>Working…</span>
      </div>
    );
  }

  return (
    <div
      role="status"
      aria-label="Working"
      data-testid="session-working-row"
      className="flex items-center gap-2 text-small-body text-subtle"
    >
      <TypingDots />
      <span aria-hidden="true" className="tabular-nums">
        {startedAt !== undefined ? (
          <>
            Working for <WorkingTimer startedAt={startedAt} />
          </>
        ) : (
          "Working…"
        )}
      </span>
    </div>
  );
}

export function SessionWorkingRowView({ row }: { row: SessionWorkingRow }) {
  const reducedMotion = usePrefersReducedMotion();
  return <WorkingIndicator startedAt={row.startedAt} reducedMotion={reducedMotion} />;
}
