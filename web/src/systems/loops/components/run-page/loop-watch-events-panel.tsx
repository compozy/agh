import { Eyebrow, formatRelativeTime } from "@agh/ui";

import { withOccurrenceKeys } from "@/lib/occurrence-keys";

import type { LoopWatchEventsState } from "../../types";

interface LoopWatchEventsPanelProps {
  state?: LoopWatchEventsState;
}

/**
 * The "Watching events" rail panel (§4.4): the parked watch-events node read-model —
 * the subscriptions the run is dormant on, the per-stream ledger cursors it will replay
 * from, and when it last woke. It renders ONLY while the run is parked on events (the
 * daemon omits `watch_events` otherwise), so an active or non-event loop shows nothing
 * (truthful UI — never a fabricated block).
 */
export function LoopWatchEventsPanel({ state }: LoopWatchEventsPanelProps) {
  if (!state || state.subscriptions.length === 0) return null;
  const cursors = Object.entries(state.cursors ?? {});
  const subscriptions = withOccurrenceKeys(
    state.subscriptions,
    subscription => `${subscription.kind}\u0000${subscription.filter ?? ""}`
  );
  return (
    <div className="border-b border-line-soft px-4 py-4" data-testid="loop-watch-events-panel">
      <Eyebrow className="mb-3 text-faint">Watching events</Eyebrow>
      <div className="flex flex-col gap-2" data-testid="loop-watch-events-subscriptions">
        {subscriptions.map(({ item: subscription, key }) => (
          <div
            key={key}
            className="rounded-md border border-line-soft bg-canvas px-2.5 py-2"
            data-testid="loop-watch-events-subscription"
          >
            <span className="block font-mono text-[11px] font-medium text-fg-strong">
              {subscription.kind}
            </span>
            {subscription.filter ? (
              <span
                className="mt-1 block font-mono text-[10.5px] leading-relaxed break-words text-muted"
                data-testid="loop-watch-events-filter"
              >
                {subscription.filter}
              </span>
            ) : (
              <span className="mt-1 block text-[10.5px] text-faint">matches every event</span>
            )}
          </div>
        ))}
      </div>
      {cursors.length > 0 ? (
        <div className="mt-3" data-testid="loop-watch-events-cursors">
          <span className="mb-1 block text-[10.5px] text-faint">Cursors</span>
          {cursors.map(([stream, seq]) => (
            <div
              key={stream}
              className="flex items-baseline justify-between gap-3 py-0.5 font-mono text-[11px]"
            >
              <span className="truncate text-subtle" title={stream}>
                {stream}
              </span>
              <span className="text-fg">{seq}</span>
            </div>
          ))}
        </div>
      ) : null}
      {state.last_wake_at ? (
        <div className="mt-3 flex items-baseline justify-between gap-3 text-[11.5px]">
          <span className="text-subtle">Last wake</span>
          <time
            dateTime={state.last_wake_at}
            className="font-medium text-fg"
            data-testid="loop-watch-events-last-wake"
          >
            {formatRelativeTime(state.last_wake_at)}
          </time>
        </div>
      ) : null}
    </div>
  );
}
