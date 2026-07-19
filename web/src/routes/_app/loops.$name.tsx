import { AlertCircle, Repeat2 } from "lucide-react";
import { Outlet, createFileRoute } from "@tanstack/react-router";

import { Empty, Spinner } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { LoopDetailView } from "@/systems/loops";
import { useLoopDetail } from "@/hooks/routes/use-loop-detail";
import { preloadLoopDetailRoute } from "./-loops-preload";

export const Route = createFileRoute("/_app/loops/$name")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: params.name, params: { name: params.name }, to: "/loops/$name" } },
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
    configQuery,
    catalogEntry,
    runsQuery,
    bindings,
    deleteLoop,
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
  if (loopQuery.isLoading || configQuery.isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="loop-detail-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }
  if (loopQuery.error || configQuery.error || !loopQuery.data) {
    return (
      <DetailState
        description={
          loopQuery.error?.message ?? configQuery.error?.message ?? `Loop ${name} not found.`
        }
        icon={AlertCircle}
        testId="loop-detail-not-found"
        title="Unable to load loop"
      />
    );
  }

  if (!configQuery.effectiveConfig) {
    return (
      <DetailState
        description="The daemon did not return effective loop configuration."
        icon={AlertCircle}
        testId="loop-detail-config-error"
        title="Unable to load loop configuration"
      />
    );
  }

  const loop = loopQuery.data;
  return (
    <LoopDetailView
      loop={loop}
      effectiveConfig={configQuery.effectiveConfig}
      graph={readGraph(loop.definition)}
      recentRuns={runsQuery.data?.runs ?? []}
      bindings={bindings.rows}
      bindingsLoading={bindings.isLoading}
      bindingJobs={bindings.jobs}
      bindingTriggers={bindings.triggers}
      successRate={catalogEntry?.success_rate_30d ?? null}
      aggregate={catalogEntry?.aggregate_30d ?? null}
      onRun={handlers.onRun}
      onConfigure={handlers.onConfigure}
      onOpenEditor={handlers.onOpenEditor}
      onDelete={handlers.onDelete}
      onDeleteReset={handlers.onDeleteReset}
      deletePending={deleteLoop.isPending}
      deleteError={deleteLoop.error?.message ?? null}
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
    <div className="flex min-h-0 flex-1 items-center justify-center py-10" data-testid={testId}>
      <Empty className="max-w-md" description={description} icon={icon} title={title} />
    </div>
  );
}
