import { ArrowUpRight } from "lucide-react";

import { Button, Eyebrow, Item, ItemActions, ItemContent, ItemFooter, ItemHeader } from "@agh/ui";

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
    <Item
      className={cn(
        "rounded-none border-b border-(--color-divider) px-4 py-3 last:border-b-0",
        className
      )}
      data-testid={`network-work-inspector-row-${entry.workId}`}
      role="listitem"
    >
      <ItemContent>
        <ItemHeader>
          <WorkChip startedAt={entry.openedAt} state={entry.state} />
          <ItemActions>
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
          </ItemActions>
        </ItemHeader>
        <p className="font-mono text-eyebrow text-(--color-text-secondary)">
          <span className="text-(--color-text-tertiary)">target </span>
          {target}
        </p>
        <ItemFooter>
          <Eyebrow weight="medium">opened {opened}</Eyebrow>
        </ItemFooter>
      </ItemContent>
    </Item>
  );
}
