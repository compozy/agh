import { AlertCircle, Repeat2 } from "lucide-react";
import { Outlet, createFileRoute } from "@tanstack/react-router";

import { Empty, Spinner } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { LoopDetailView } from "@/systems/loops";
import { useLoopDetail } from "@/hooks/routes/use-loop-detail";
import { preloadLoopDetailRoute } from "./-loops-preload";

export const Route = createFileRoute("/_app/loops/$name")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: params.name, icon: Repeat2 },
  }),
  loader: ({ context, location, params }) =>
    location.pathname.split("/").filter(Boolean).length === 2
      ? preloadLoopDetailRoute(context.queryClient, params.name)
      : Promise.resolve(),
  component: LoopDetailRoute,
});

function LoopDetailRoute() {
  const { name } = Route.useParams();
  const {
    hasChildMatch,
    workspaceId,
    loopQuery,
    catalogEntry,
    runsQuery,
    bindings,
    readGraph,
    handlers,
  } = useLoopDetail(name);

  if (hasChildMatch) {
    return <Outlet />;
  }
  if (workspaceId === "") {
    return (
      <DetailState
        description="Select a workspace to inspect this Loop."
        testId="loop-detail-no-workspace"
        title="No workspace selected"
      />
    );
  }
  if (loopQuery.isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="loop-detail-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }
  if (loopQuery.error || !loopQuery.data) {
    return (
      <DetailState
        description={loopQuery.error?.message ?? `Loop ${name} not found.`}
        icon={AlertCircle}
        testId="loop-detail-not-found"
        title="Unable to load loop"
      />
    );
  }

  const loop = loopQuery.data;
  return (
    <LoopDetailView
      loop={loop}
      graph={readGraph(loop.definition)}
      recentRuns={runsQuery.data?.runs ?? []}
      bindings={bindings.rows}
      bindingsLoading={bindings.isLoading}
      bindingJobs={bindings.jobs}
      bindingTriggers={bindings.triggers}
      successRate={catalogEntry?.success_rate_30d ?? null}
      aggregate={catalogEntry?.aggregate_30d ?? null}
      onBack={handlers.onBack}
      onRun={handlers.onRun}
      onConfigure={handlers.onConfigure}
      onFork={handlers.onFork}
      onAddTrigger={handlers.onAddTrigger}
      onAddSchedule={handlers.onAddSchedule}
    />
  );
}

interface DetailStateProps {
  title: string;
  description: string;
  testId: string;
  icon?: typeof Repeat2;
}

function DetailState({ title, description, testId, icon = Repeat2 }: DetailStateProps) {
  return (
    <div
      className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
      data-testid={testId}
    >
      <Empty className="max-w-md" description={description} icon={icon} title={title} />
    </div>
  );
}
