import { useState } from "react";
import { Activity, AlertCircle } from "lucide-react";
import { Outlet, createFileRoute, useChildMatches } from "@tanstack/react-router";

import { Empty, ListingPage, Spinner } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { LoopRunsView, type LoopOutcomeValue, useLoopRuns } from "@/systems/loops";
import { useActiveWorkspace } from "@/systems/workspace";

export const Route = createFileRoute("/_app/loop-runs")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Runs", icon: Activity },
  }),
  component: LoopRunsRoute,
});

function LoopRunsRoute() {
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const workspaceLabel = activeWorkspace?.name ?? activeWorkspace?.id ?? "workspace";
  const [outcome, setOutcome] = useState<LoopOutcomeValue>("all");
  // Skip the runs fetch while the run-detail child route owns the view.
  const runsQuery = useLoopRuns(workspaceId, {}, workspaceId !== "" && !hasChildMatch);

  if (hasChildMatch) {
    return <Outlet />;
  }

  if (workspaceId === "") {
    return (
      <RunsState
        description="Select a workspace to view its Loop runs."
        testId="loop-runs-no-workspace"
        title="No workspace selected"
      />
    );
  }

  if (runsQuery.isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="loop-runs-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }

  if (runsQuery.error) {
    return (
      <RunsState
        description={runsQuery.error.message ?? "Failed to load loop runs"}
        icon={AlertCircle}
        testId="loop-runs-error"
        title="Unable to load runs"
      />
    );
  }

  const runs = runsQuery.data?.runs ?? [];

  if (runs.length === 0) {
    return (
      <RunsState
        description="No Loop has run in this workspace yet."
        testId="loop-runs-empty"
        title="No runs yet"
      />
    );
  }

  return (
    <ListingPage data-testid="loop-runs">
      <ListingPage.Head
        count={runs.length}
        countTestId="loop-runs-page-count"
        meta={
          <>
            <span>Every execution of a Loop, across the full outcome spectrum.</span>
            <ListingPage.MetaDot />
            <span>{workspaceLabel}</span>
          </>
        }
        title="Runs"
      />
      <LoopRunsView onOutcomeChange={setOutcome} outcome={outcome} runs={runs} />
    </ListingPage>
  );
}

interface RunsStateProps {
  title: string;
  description: string;
  testId: string;
  icon?: typeof Activity;
}

function RunsState({ title, description, testId, icon = Activity }: RunsStateProps) {
  return (
    <div
      className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
      data-testid={testId}
    >
      <Empty className="max-w-md" description={description} icon={icon} title={title} />
    </div>
  );
}
