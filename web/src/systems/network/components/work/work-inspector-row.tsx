import { ArrowUpRight } from "lucide-react";

import { Button } from "@agh/ui";

import { cn } from "@/lib/utils";

import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { OpenWorkEntry } from "../../hooks/use-work";
import { WorkChip } from "./work-chip";

export interface WorkInspectorRowProps {
  entry: OpenWorkEntry;
  onJump?: (entry: OpenWorkEntry) => void;
  className?: string;
}

export function WorkInspectorRow({ entry, onJump, className }: WorkInspectorRowProps) {
  const opened = formatNetworkRelativeTime(entry.openedAt);
  const target = entry.targetPeerId ?? "unassigned";

  return (
    <li
      className={cn(
        "flex flex-col gap-1.5 border-b border-(--color-divider) px-4 py-3 last:border-b-0",
        className
      )}
      data-testid={`network-work-inspector-row-${entry.workId}`}
    >
      <div className="flex items-center justify-between gap-2">
        <WorkChip startedAt={entry.openedAt} state={entry.state} />
        <Button
          aria-label="Jump to message"
          data-testid={`network-work-inspector-jump-${entry.workId}`}
          onClick={onJump ? () => onJump(entry) : undefined}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <ArrowUpRight aria-hidden="true" className="size-3.5" />
        </Button>
      </div>
      <p className="font-mono text-eyebrow text-(--color-text-secondary)">
        <span className="text-(--color-text-tertiary)">target </span>
        {target}
      </p>
      <p className="font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)">
        opened {opened}
      </p>
    </li>
  );
}
