import { Empty } from "@agh/ui";
import { Activity } from "lucide-react";

import { cn } from "@/lib/utils";

import type { OpenWorkEntry } from "../../hooks/use-work";
import { WorkInspectorRow } from "./work-inspector-row";

export interface WorkInspectorProps {
  entries: ReadonlyArray<OpenWorkEntry>;
  isLoading?: boolean;
  onJump?: (entry: OpenWorkEntry) => void;
  className?: string;
}

export function WorkInspector({
  entries,
  isLoading = false,
  onJump,
  className,
}: WorkInspectorProps) {
  return (
    <section
      aria-label="Open network work"
      className={cn("flex min-h-0 flex-1 flex-col", className)}
      data-testid="network-work-inspector"
    >
      <header className="flex items-baseline justify-between border-b border-[color:var(--color-divider)] px-4 py-3">
        <h2 className="text-[14px] font-semibold text-[color:var(--color-text-primary)]">Work</h2>
        <span
          className="font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
          data-testid="network-work-inspector-count"
        >
          {entries.length} open
        </span>
      </header>

      {isLoading && entries.length === 0 ? (
        <p className="px-4 py-6 text-[13px] text-[color:var(--color-text-tertiary)]">Loading…</p>
      ) : entries.length === 0 ? (
        <div className="px-4 py-6">
          <Empty
            className="max-w-sm"
            description="The active container has no open work right now."
            fill={false}
            icon={Activity}
            title="No work in flight."
          />
        </div>
      ) : (
        <ul
          aria-label="Open work entries"
          className="flex flex-1 flex-col overflow-y-auto"
          data-testid="network-work-inspector-list"
        >
          {entries.map(entry => (
            <WorkInspectorRow entry={entry} key={entry.workId} onJump={onJump} />
          ))}
        </ul>
      )}
    </section>
  );
}
