import { useReducer } from "react";

import { toast } from "@agh/ui";
import {
  applyLoopEventFrame,
  buildRunMeters,
  buildRunTimeline,
  emptyLoopRunLiveState,
  isTerminalLoopStatus,
  latestGenerationBreadth,
  mergeGoalTurnTimeline,
  type LoopGateDecision,
  useApproveLoopRun,
  useLoop,
  useLoopRun,
  useLoopStream,
  useGoalTurns,
  usePauseLoopRun,
  useResumeLoopRun,
  useStopLoopRun,
} from "@/systems/loops";

/**
 * The live run-page view-model (§4.4): composes the run projection (`getLoopRun`) with
 * the loop definition (`getLoop`, for the contract/graph the run projection omits) and
 * an SSE reducer over the forward event contract. Meters, timeline, and breadth derive
 * from daemon truth; the live token count overlays the polled run when fresher. The
 * operator controls and the gate decision go through the sanctioned mutation hooks.
 */
export function useLoopRunPage(workspaceId: string, runId: string) {
  const enabled = workspaceId !== "" && runId !== "";
  const runQuery = useLoopRun(workspaceId, runId, enabled);
  const run = runQuery.data?.run;
  const generations = runQuery.data?.generations;
  const watchEvents = runQuery.data?.watch_events ?? undefined;
  const executedDefinition = runQuery.data?.executed_definition;
  const loopName = run?.loop_name ?? "";
  const loopQuery = useLoop(
    workspaceId,
    loopName,
    enabled && loopName !== "" && !executedDefinition
  );
  const definition = executedDefinition ?? loopQuery.data?.definition;

  const [live, dispatch] = useReducer(applyLoopEventFrame, undefined, emptyLoopRunLiveState);
  const isLive = runQuery.isSuccess && !isTerminalLoopStatus(run?.status);
  useLoopStream(workspaceId, runId, { enabled: isLive, onEvent: dispatch });
  const goalTurnsQuery = useGoalTurns(workspaceId, runId, { enabled: runQuery.isSuccess });
  const goalTurns = mergeGoalTurnTimeline(goalTurnsQuery.data?.turns ?? [], live.goalTurns);

  const pauseMutation = usePauseLoopRun();
  const resumeMutation = useResumeLoopRun();
  const stopMutation = useStopLoopRun();
  const approveMutation = useApproveLoopRun();

  const breadth = latestGenerationBreadth(generations);
  // A lifecycle refetch can return tokens newer than the latest streamed tick;
  // take the max so the meter never steps backward between a tick and a poll.
  const effectiveRun =
    run && live.tokensUsed !== null
      ? { ...run, tokens_used: Math.max(run.tokens_used, live.tokensUsed) }
      : run;
  const meters = effectiveRun ? buildRunMeters(effectiveRun, breadth) : [];
  const timeline = buildRunTimeline(generations, definition);

  const handlePause = () => {
    pauseMutation.mutate(
      { workspaceId, runId },
      {
        onSuccess: () => toast.success("Pause requested — pausing at the next generation boundary"),
        onError: error =>
          toast.error(error instanceof Error ? error.message : "Failed to pause run"),
      }
    );
  };

  const handleResume = () => {
    resumeMutation.mutate(
      { workspaceId, runId },
      {
        onSuccess: () => toast.success("Run resumed"),
        onError: error =>
          toast.error(error instanceof Error ? error.message : "Failed to resume run"),
      }
    );
  };

  const handleStop = () => {
    stopMutation.mutate(
      { workspaceId, runId },
      {
        onSuccess: () => toast.success("Run stopped"),
        onError: error =>
          toast.error(error instanceof Error ? error.message : "Failed to stop run"),
      }
    );
  };

  const handleDecision = (decision: LoopGateDecision, gateId: string) => {
    approveMutation.mutate(
      { workspaceId, runId, data: { decision, gate_id: gateId } },
      {
        onSuccess: () => toast.success(`Gate decision recorded — ${decision}`),
        onError: error =>
          toast.error(error instanceof Error ? error.message : "Failed to record gate decision"),
      }
    );
  };

  return {
    runQuery,
    run,
    watchEvents,
    definition,
    contract: definition?.contract,
    loopVersion: run?.definition_version ?? loopQuery.data?.version,
    live,
    goalTurns,
    goalTurnsQuery,
    isLive,
    meters,
    timeline,
    handlePause,
    handleResume,
    handleStop,
    handleDecision,
    isPausePending: pauseMutation.isPending,
    isResumePending: resumeMutation.isPending,
    isStopPending: stopMutation.isPending,
    isApprovePending: approveMutation.isPending,
  };
}
