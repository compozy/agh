import { GitBranch } from "lucide-react";

import { Button, Empty, Spinner } from "@agh/ui";

import type { LoopChannelMessage, LoopGateVerdict } from "../../lib/loop-events";
import type { LoopTimelineGeneration } from "../../lib/loop-timeline";
import type { GoalTurnTimelineItem } from "../../hooks/use-goal-turns";
import { LoopGenerationCard } from "./loop-generation-card";

const EMPTY_GOAL_TURNS: readonly GoalTurnTimelineItem[] = [];

interface LoopGenerationTimelineProps {
  generations: LoopTimelineGeneration[];
  gateVerdicts: Record<string, LoopGateVerdict>;
  channelMessages: LoopChannelMessage[];
  isLive: boolean;
  goalTurns?: readonly GoalTurnTimelineItem[];
  hasMoreGoalTurns?: boolean;
  isLoadingMoreGoalTurns?: boolean;
  onLoadMoreGoalTurns?: () => void;
}

/**
 * The generation-by-generation timeline (§4.4): collapsible generation cards on a flat
 * node spine, newest-first. Each carries carried-forward read-only tags, gate verdict
 * cards, the `awaiting_child` child-run link, and the embedded channel — all rendered
 * from daemon truth (the run's generation outputs + streamed events).
 */
export function LoopGenerationTimeline({
  generations,
  gateVerdicts,
  channelMessages,
  isLive,
  goalTurns = EMPTY_GOAL_TURNS,
  hasMoreGoalTurns = false,
  isLoadingMoreGoalTurns = false,
  onLoadMoreGoalTurns,
}: LoopGenerationTimelineProps) {
  if (generations.length === 0) {
    return (
      <Empty
        className="py-10"
        icon={GitBranch}
        title="No generations yet"
        description="Generations appear here as the loop iterates."
      />
    );
  }
  return (
    <div className="flex flex-col gap-3" data-testid="loop-generation-timeline">
      {generations.map(generation => (
        <LoopGenerationCard
          key={generation.generation}
          generation={generation}
          gateVerdicts={gateVerdicts}
          channelMessages={channelMessages}
          isLive={isLive}
          goalTurns={goalTurns}
        />
      ))}
      {hasMoreGoalTurns && onLoadMoreGoalTurns ? (
        <Button
          className="self-center"
          disabled={isLoadingMoreGoalTurns}
          onClick={onLoadMoreGoalTurns}
          size="sm"
          type="button"
          variant="ghost"
        >
          {isLoadingMoreGoalTurns ? <Spinner className="size-3" /> : null}
          Load more Goal turns
        </Button>
      ) : null}
    </div>
  );
}
