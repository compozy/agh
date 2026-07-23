import { ChevronDown } from "lucide-react";

import { Button, Collapsible, CollapsibleContent, CollapsibleTrigger, Spinner } from "@agh/ui";

import type { GoalTurnTimelineItem } from "../../hooks/use-goal-turns";
import { GoalTurnTimeline } from "./goal-turn-timeline";

interface LoopRunTurnsDisclosureProps {
  turns: readonly GoalTurnTimelineItem[];
  live: boolean;
  hasMore?: boolean;
  isLoadingMore?: boolean;
  onLoadMore?: () => void;
}

/**
 * The Turns disclosure (§3): goal-node story rows fold their `/turns` audit
 * behind a quiet toggle instead of embedding it inline. Renders nothing when
 * the node has no turns yet — never an empty shell.
 */
export function LoopRunTurnsDisclosure({
  turns,
  live,
  hasMore = false,
  isLoadingMore = false,
  onLoadMore,
}: LoopRunTurnsDisclosureProps) {
  if (turns.length === 0 && !hasMore) return null;
  return (
    <Collapsible data-testid="loop-run-turns-disclosure">
      <CollapsibleTrigger className="group mt-2 inline-flex items-center gap-1.5 rounded-xs text-badge font-medium text-muted hover:text-fg-strong">
        Turns
        <span className="font-mono text-mono-id tabular-nums text-faint">{turns.length}</span>
        <ChevronDown
          aria-hidden="true"
          className="size-3 transition-transform group-data-[state=closed]:-rotate-90"
        />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <GoalTurnTimeline className="mt-3" live={live} turns={turns} />
        {hasMore && onLoadMore ? (
          <Button
            className="mt-2"
            disabled={isLoadingMore}
            onClick={onLoadMore}
            size="sm"
            type="button"
            variant="ghost"
          >
            {isLoadingMore ? <Spinner className="size-3" /> : null}
            Load more turns
          </Button>
        ) : null}
      </CollapsibleContent>
    </Collapsible>
  );
}
