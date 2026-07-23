import { useEffect, useReducer, useState } from "react";

import { useNavigate } from "@tanstack/react-router";

import { toast } from "@agh/ui";

import {
  applyLoopEventFrame,
  buildInputRows,
  buildNextNote,
  buildRunProgress,
  buildRunStory,
  buildRunUsage,
  emptyLoopRunLiveState,
  findWatchNode,
  formatClockDuration,
  goalNodeIds,
  humanizeStartOrigin,
  isTerminalLoopStatus,
  latestGateVerdict,
  mergeGoalTurnTimeline,
  readLoopGraph,
  runElapsedSeconds,
  usageNote,
  usageSnapshotFacts,
  watchedSubject,
  type LoopGateDecision,
  type LoopRunEventFrame,
  useApproveLoopRun,
  useGoalTurns,
  useLoop,
  useLoopRun,
  useLoopStream,
  usePauseLoopRun,
  useResumeLoopRun,
  useRunLoop,
  useStopLoopRun,
} from "@/systems/loops";

/** Statuses whose main column leads with the "Happening now" card (§7). */
const NOW_CARD_STATUSES = new Set(["running", "watching", "paused"]);

/** One shared 1s clock while the run ticks; parked spans are timestamp-frozen. */
function useNowTick(enabled: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!enabled) return undefined;
    setNow(Date.now());
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [enabled]);
  return now;
}

/** The `from` side of the transition that landed the run on `toStatus`. */
function findTransitionFrom(
  frames: readonly LoopRunEventFrame[],
  toStatus: string
): string | undefined {
  for (let index = frames.length - 1; index >= 0; index--) {
    const frame = frames[index];
    if (frame.kind !== "status_changed") continue;
    const payload = frame.payload as Record<string, unknown> | null;
    if (!payload || typeof payload !== "object") continue;
    const to = typeof payload.to === "string" ? payload.to : payload.status;
    if (to !== toStatus) continue;
    return typeof payload.from === "string" && payload.from !== "" ? payload.from : undefined;
  }
  return undefined;
}

/**
 * The live run-page view-model (redesign spec §2-§5): composes the run
 * projection (`getLoopRun`) with the pinned definition and the SSE reducer over
 * the forward event contract, then derives the story timeline, the group
 * progress, the usage rows, and the About projections. The live token count
 * overlays the polled run when fresher; controls and the gate decision go
 * through the sanctioned mutation hooks.
 */
export function useLoopRunPage(workspaceId: string, runId: string) {
  const navigate = useNavigate();
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
  // Terminal runs replay their retained status event once so generation-zero
  // failures remain visible after navigation or reload; the hook closes on it.
  useLoopStream(workspaceId, runId, { enabled: runQuery.isSuccess, onEvent: dispatch });
  const goalTurnsQuery = useGoalTurns(workspaceId, runId, { enabled: runQuery.isSuccess });
  const goalTurns = mergeGoalTurnTimeline(goalTurnsQuery.data?.turns ?? [], live.goalTurns);

  const pauseMutation = usePauseLoopRun();
  const resumeMutation = useResumeLoopRun();
  const stopMutation = useStopLoopRun();
  const approveMutation = useApproveLoopRun();
  const runLoopMutation = useRunLoop();

  // A lifecycle refetch can return tokens newer than the latest streamed tick;
  // take the max so Usage never steps backward between a tick and a poll.
  const effectiveRun =
    run && live.tokensUsed !== null
      ? { ...run, tokens_used: Math.max(run.tokens_used, live.tokensUsed) }
      : run;

  const graph = definition ? readLoopGraph(definition) : null;
  const story = buildRunStory(live.frames, {
    status: run?.status,
    reattemptStrategy: run?.reattempt_strategy,
    graph,
    generations,
  });

  const nowMs = useNowTick(run?.status === "running");
  const elapsedSeconds = run ? runElapsedSeconds(run, nowMs) : 0;
  const stepStartedMs =
    run?.status === "running" && story.now ? Date.parse(story.now.startedAt) : Number.NaN;
  const stepElapsedLabel = Number.isNaN(stepStartedMs)
    ? null
    : formatClockDuration((nowMs - stepStartedMs) / 1000);

  const latestVerdict = latestGateVerdict(live.gateVerdicts);
  const openPoints = latestVerdict ? latestVerdict.blockingIssues.length : null;
  const progress = effectiveRun ? buildRunProgress(effectiveRun, generations, openPoints) : null;
  const usageRows = effectiveRun ? buildRunUsage(effectiveRun, elapsedSeconds) : [];
  const note = run ? usageNote(run) : null;
  const approvalFallbackFacts = effectiveRun
    ? usageSnapshotFacts(effectiveRun, elapsedSeconds)
    : [];

  const subject = run ? watchedSubject(run, definition) : null;
  const inputRows = run ? buildInputRows(run, definition) : [];
  const startedBy = run ? humanizeStartOrigin(run) : "";
  const watchNode = graph ? findWatchNode(graph) : null;
  const watchPoll = typeof watchNode?.watch?.poll === "string" ? watchNode.watch.poll : "";
  const watchCadence = watchPoll ? `checks every ${watchPoll}` : null;
  const goalIds = graph ? goalNodeIds(graph) : new Set<string>();
  const showNextNote =
    run?.status === "running" || run?.status === "watching" || run?.status === "queued";
  const nextNote = showNextNote ? buildNextNote(graph, generations) : null;
  const showNowCard =
    run !== undefined &&
    (NOW_CARD_STATUSES.has(run.status) ||
      (run.status === "queued" && definition?.concurrency === "queue"));
  const terminalFromStatus = run ? findTransitionFrom(live.frames, run.status) : undefined;

  const handlePause = () => {
    pauseMutation.mutate(
      { workspaceId, runId },
      {
        onSuccess: () => toast.success("Pause requested — the run parks at the next safe point"),
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
        onSuccess: () => toast.success("Decision recorded"),
        onError: error =>
          toast.error(error instanceof Error ? error.message : "Failed to record the decision"),
      }
    );
  };

  const handleStartNewRun = () => {
    if (!run) return;
    runLoopMutation.mutate(
      { workspaceId, name: run.loop_name, data: { inputs: run.inputs } },
      {
        onSuccess: result => {
          const newRunId = result.run?.id;
          if (newRunId) {
            toast.success("New run started");
            void navigate({ to: "/loop-runs/$runId", params: { runId: newRunId } });
            return;
          }
          toast.error("The daemon accepted the request but returned no run");
        },
        onError: error =>
          toast.error(error instanceof Error ? error.message : "Failed to start a new run"),
      }
    );
  };

  const version = run?.definition_version ?? loopQuery.data?.version;
  const versionLabel =
    version !== undefined
      ? executedDefinition
        ? `v${version} · pinned`
        : `v${version}`
      : undefined;
  const pendingAction = approveMutation.isPending
    ? ("approve" as const)
    : runLoopMutation.isPending
      ? ("start-new-run" as const)
      : undefined;

  return {
    runQuery,
    run,
    effectiveRun,
    definition,
    graph,
    watchEvents,
    versionLabel,
    live,
    story,
    goalIds,
    goalTurns,
    goalTurnsQuery,
    isLive,
    progress,
    usageRows,
    usageNote: note,
    approvalFallbackFacts,
    latestVerdict,
    subject,
    hasWatchSource: watchNode !== null,
    watchCadence,
    inputRows,
    startedBy,
    elapsedLabel: formatClockDuration(elapsedSeconds),
    stepElapsedLabel,
    nextNote,
    showNowCard,
    terminalFromStatus,
    handlePause,
    handleResume,
    handleStop,
    handleDecision,
    handleStartNewRun,
    pendingAction,
    isPausePending: pauseMutation.isPending,
    isResumePending: resumeMutation.isPending,
    isStopPending: stopMutation.isPending,
  };
}
