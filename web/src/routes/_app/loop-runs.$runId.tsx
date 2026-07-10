import { Activity, AlertCircle } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Empty, Spinner } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { useLoopRunPage } from "@/hooks/routes/use-loop-run-page";
import {
  LoopApprovalGate,
  LoopGenerationTimeline,
  LoopRunContractHeader,
  LoopRunControls,
  LoopRunEventsRail,
  LoopRunFacts,
  LoopRunMeters,
  LoopStatusLegend,
  LoopWatchEventsPanel,
} from "@/systems/loops";
import { useActiveWorkspace } from "@/systems/workspace";

export const Route = createFileRoute("/_app/loop-runs/$runId")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `Run ${params.runId}`, icon: Activity },
  }),
  component: LoopRunDetailRoute,
});

function LoopRunDetailRoute() {
  const { runId } = Route.useParams();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";

  if (workspaceId === "") {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="loop-run-detail-no-workspace"
      >
        <Empty
          className="max-w-md"
          description="Select a workspace to monitor this run."
          icon={Activity}
          title="No workspace selected"
        />
      </div>
    );
  }

  // Key by runId so the SSE reducer state resets cleanly on a run switch.
  return <LoopRunDetail key={runId} workspaceId={workspaceId} runId={runId} />;
}

interface LoopRunDetailProps {
  workspaceId: string;
  runId: string;
}

function LoopRunDetail({ workspaceId, runId }: LoopRunDetailProps) {
  const page = useLoopRunPage(workspaceId, runId);

  if (page.runQuery.isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="loop-run-detail-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.runQuery.error || !page.run) {
    return (
      <div
        className="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 px-6 text-center"
        data-testid="loop-run-detail-not-found"
      >
        <AlertCircle className="size-6 text-danger" />
        <p className="text-sm text-muted">
          {page.runQuery.error?.message || `Run ${runId} not found.`}
        </p>
      </div>
    );
  }

  const { run } = page;
  return (
    <div className="flex min-h-0 flex-1" data-testid="loop-run-detail-content">
      <div className="flex min-w-0 flex-1 flex-col overflow-y-auto">
        <LoopRunContractHeader
          run={run}
          contract={page.contract}
          controls={
            <LoopRunControls
              status={run.status}
              pauseRequested={run.pause_requested}
              isPausePending={page.isPausePending}
              isResumePending={page.isResumePending}
              isStopPending={page.isStopPending}
              onPause={page.handlePause}
              onResume={page.handleResume}
              onStop={page.handleStop}
            />
          }
          meters={<LoopRunMeters meters={page.meters} />}
        />
        <div className="flex flex-col gap-4 px-6 py-5">
          {run.status === "needs-approval" ? (
            <LoopApprovalGate
              run={run}
              request={page.live.needsApproval}
              isPending={page.isApprovePending}
              onDecision={page.handleDecision}
            />
          ) : null}
          <LoopGenerationTimeline
            generations={page.timeline}
            gateVerdicts={page.live.gateVerdicts}
            channelMessages={page.live.channelMessages}
            isLive={page.isLive}
          />
        </div>
      </div>
      <aside
        className="hidden w-[332px] shrink-0 overflow-y-auto border-l border-line bg-canvas-soft xl:block"
        data-testid="loop-run-detail-rail"
      >
        <LoopRunEventsRail events={page.live.events} isLive={page.isLive} />
        <LoopWatchEventsPanel state={page.watchEvents} />
        <LoopRunFacts
          run={run}
          loopVersion={page.loopVersion}
          concurrency={page.definition?.concurrency}
        />
        <LoopStatusLegend />
      </aside>
    </div>
  );
}
