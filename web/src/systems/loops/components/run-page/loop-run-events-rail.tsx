import { Eyebrow, PillDot } from "@agh/ui";

import type { LoopLiveEvent } from "../../lib/loop-events";

interface LoopRunEventsRailProps {
  events: LoopLiveEvent[];
  isLive: boolean;
}

const TONE_CLASS: Record<LoopLiveEvent["tone"], string> = {
  neutral: "text-subtle",
  ok: "text-success",
  warn: "text-warning",
  err: "text-danger",
};

/** `HH:MM` from an ISO timestamp; blank when unparseable. */
function clock(at: string): string {
  const parsed = Date.parse(at);
  if (Number.isNaN(parsed)) return "";
  const date = new Date(parsed);
  return `${String(date.getHours()).padStart(2, "0")}:${String(date.getMinutes()).padStart(2, "0")}`;
}

/**
 * The "Live events" rail (§4.4): the streamed SSE frames, newest-first. The daemon
 * emits only `status_changed` today; the rail fills as the forward event kinds ship.
 */
export function LoopRunEventsRail({ events, isLive }: LoopRunEventsRailProps) {
  return (
    <div className="border-b border-line-soft px-4 py-4" data-testid="loop-run-events">
      <div className="mb-3 flex items-center gap-2">
        <Eyebrow className="text-faint">Live events</Eyebrow>
        {isLive ? (
          <span className="ml-auto inline-flex items-center gap-1.5 text-[10px] font-medium text-accent-strong">
            <PillDot tone="accent" pulse size="sm" />
            streaming
          </span>
        ) : null}
      </div>
      {events.length === 0 ? (
        <p className="text-[11.5px] text-subtle" data-testid="loop-run-events-empty">
          No events streamed yet.
        </p>
      ) : (
        <div className="flex flex-col gap-0.5">
          {events.map(event => (
            <div
              key={event.seq}
              className="grid grid-cols-[38px_96px_1fr] gap-2 py-0.5 font-mono text-[11px] leading-relaxed"
              data-testid="loop-run-event"
            >
              <span className="text-faint">{clock(event.at)}</span>
              <span className={`truncate ${TONE_CLASS[event.tone]}`} title={event.kind}>
                {event.kind}
              </span>
              <span className="truncate text-muted" title={event.message}>
                {event.message}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
