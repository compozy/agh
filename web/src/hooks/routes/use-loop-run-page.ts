import { useCallback, useMemo, useReducer } from "react";

import { toast } from "@agh/ui";
import {
  applyLoopEventFrame,
  buildRunMeters,
  buildRunTimeline,
  emptyLoopRunLiveState,
  isTerminalLoopStatus,
  latestGenerationBreadth,
  type LoopGateDecision,
  useApproveLoopRun,
  useLoop,
  useLoopRun,
  useLoopStream,
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

  const pauseMutation = usePauseLoopRun();
  const resumeMutation = useResumeLoopRun();
  const stopMutation = useStopLoopRun();
  const approveMutation = useApproveLoopRun();

  const breadth = useMemo(() => latestGenerationBreadth(generations), [generations]);
  const meters = useMemo(() => {
    if (!run) return [];
    // A lifecycle refetch can return a `tokens_used` newer than the last `token_tick`;
    // take the max so the Tokens/Cost meters never step backward to a stale streamed
    // value between a tick and the next poll (R-005).
    const effective =
      live.tokensUsed !== null
        ? { ...run, tokens_used: Math.max(run.tokens_used, live.tokensUsed) }
        : run;
    return buildRunMeters(effective, breadth);
  }, [run, live.tokensUsed, breadth]);
  const timeline = useMemo(
    () => buildRunTimeline(generations, definition),
    [generations, definition]
  );

  const handlePause = useCallback(() => {
    pauseMutation.mutate(
      { workspaceId, runId },
      {
        onSuccess: () => toast.success("Pause requested — pausing at the next generation boundary"),
        onError: error =>
          toast.error(error instanceof Error ? error.message : "Failed to pause run"),
      }
    );
  }, [pauseMutation, workspaceId, runId]);

  const handleResume = useCallback(() => {
    resumeMutation.mutate(
      { workspaceId, runId },
      {
        onSuccess: () => toast.success("Run resumed"),
        onError: error =>
          toast.error(error instanceof Error ? error.message : "Failed to resume run"),
      }
    );
  }, [resumeMutation, workspaceId, runId]);

  const handleStop = useCallback(() => {
    stopMutation.mutate(
      { workspaceId, runId },
      {
        onSuccess: () => toast.success("Run stopped"),
        onError: error =>
          toast.error(error instanceof Error ? error.message : "Failed to stop run"),
      }
    );
  }, [stopMutation, workspaceId, runId]);

  const handleDecision = useCallback(
    (decision: LoopGateDecision, gateId: string) => {
      approveMutation.mutate(
        { workspaceId, runId, data: { decision, gate_id: gateId } },
        {
          onSuccess: () => toast.success(`Gate decision recorded — ${decision}`),
          onError: error =>
            toast.error(error instanceof Error ? error.message : "Failed to record gate decision"),
        }
      );
    },
    [approveMutation, workspaceId, runId]
  );

  return {
    runQuery,
    run,
    watchEvents,
    definition,
    contract: definition?.contract,
    loopVersion: run?.definition_version ?? loopQuery.data?.version,
    live,
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
