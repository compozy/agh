import { ChevronRight } from "lucide-react";

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
  Empty,
  Eyebrow,
  Panel,
  Pill,
  Section,
  Time,
} from "@agh/ui";

import { homeActivityTone, isQuietActivityEvent } from "../lib/activity-classes";
import type { HomeActivityEvent } from "../types";

export interface HomeActivityFeedProps {
  events: HomeActivityEvent[];
}

const LOUD_EVENT_LIMIT = 8;
const RECENT_WINDOW_MS = 60 * 60 * 1000;

function eventTitle(event: HomeActivityEvent): string {
  const summary = event.summary?.trim();
  if (summary) {
    return summary;
  }
  return event.type;
}

function toneFor(event: HomeActivityEvent) {
  const tone = homeActivityTone(event);
  return tone === "neutral" ? undefined : tone;
}

function ActivityRow({ event, quiet }: { event: HomeActivityEvent; quiet?: boolean }) {
  const tone = toneFor(event);
  return (
    <div
      className="grid grid-cols-[14px_minmax(0,1fr)_auto] items-start gap-2.5 px-4 py-2.5"
      data-slot="home-activity-row"
    >
      <span className="mt-1.5 flex justify-center">
        {tone ? (
          <Pill.Dot tone={tone} />
        ) : (
          <span aria-hidden="true" className="size-1.5 rounded-full bg-faint" />
        )}
      </span>
      <span
        className={
          quiet
            ? "min-w-0 truncate text-small-body text-muted"
            : "min-w-0 truncate text-small-body font-medium text-fg-strong"
        }
      >
        {event.agent_name ? <span className="font-medium">{event.agent_name}</span> : null}
        {event.agent_name ? " · " : ""}
        <span className={quiet ? undefined : "font-normal text-muted"}>{eventTitle(event)}</span>
      </span>
      <span className="font-mono text-mono-id tabular-nums text-subtle">
        <Time iso={event.timestamp} />
      </span>
    </div>
  );
}

/**
 * Zone 5b — the recent event stream. Lifecycle events stay loud; per-call tool
 * traffic and config reads fold behind an explicit disclosure so the feed
 * stays honest without shouting.
 */
export function HomeActivityFeed({ events }: HomeActivityFeedProps) {
  const loud = events.filter(event => !isQuietActivityEvent(event)).slice(0, LOUD_EVENT_LIMIT);
  const quiet = events.filter(event => isQuietActivityEvent(event));

  const now = Date.now();
  const separatorIndex = loud.findIndex(event => {
    const at = Date.parse(event.timestamp);
    return !Number.isNaN(at) && now - at > RECENT_WINDOW_MS;
  });

  return (
    <Section
      bodyClassName="flex flex-1 flex-col"
      className="flex min-h-full flex-col"
      data-slot="home-activity"
      label="Activity"
    >
      <Panel bodyClassName="flex flex-1 flex-col p-0" className="flex-1 overflow-hidden">
        {loud.length === 0 && quiet.length === 0 ? (
          <Empty description="Events land here as agents work." title="No recent activity" />
        ) : (
          <div className="flex flex-1 flex-col divide-y divide-line-soft">
            {loud.map((event, index) => (
              <div key={event.id}>
                {index === separatorIndex && index > 0 ? (
                  <div className="flex items-center gap-2.5 border-b border-line-soft px-4 py-1.5">
                    <Eyebrow className="text-faint">Earlier today</Eyebrow>
                    <span aria-hidden="true" className="h-px flex-1 bg-line-soft" />
                  </div>
                ) : null}
                <ActivityRow event={event} />
              </div>
            ))}
            {quiet.length > 0 ? (
              <Collapsible>
                <CollapsibleTrigger className="group flex w-full items-center gap-2 px-4 py-2.5 text-left text-small-body text-subtle transition-colors duration-base hover:bg-row-hover hover:text-muted focus-visible:shadow-focus-inset focus-visible:outline-none">
                  <ChevronRight
                    aria-hidden="true"
                    className="size-3.5 text-faint transition-transform duration-base group-data-[panel-open]:rotate-90"
                  />
                  {quiet.length} quieter event{quiet.length === 1 ? "" : "s"} — tool calls and
                  config reads
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <div className="divide-y divide-line-soft border-t border-line-soft">
                    {quiet.map(event => (
                      <ActivityRow event={event} key={event.id} quiet />
                    ))}
                  </div>
                </CollapsibleContent>
              </Collapsible>
            ) : null}
          </div>
        )}
      </Panel>
    </Section>
  );
}
