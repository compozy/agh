import { AlertCircle, Repeat2 } from "lucide-react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { Empty, Spinner } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { LoopRunForm, useLoop } from "@/systems/loops";
import { useActiveWorkspace } from "@/systems/workspace";
import { preloadLoopRunFormRoute } from "./-loops-preload";

export const Route = createFileRoute("/_app/loops/$name/run")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `Run ${params.name}`, icon: Repeat2 },
  }),
  loader: ({ context, params }) => preloadLoopRunFormRoute(context.queryClient, params.name),
  component: LoopRunFormRoute,
});

/**
 * Run-form entry for a Loop (design §4.3): the auto-generated typed input form, the
 * Advanced per-run overrides, the live contract preview, and Dry run / Run. On a
 * successful Run the started run's id routes to its live run page.
 */
function LoopRunFormRoute() {
  const { name } = Route.useParams();
  const navigate = useNavigate();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const loopQuery = useLoop(workspaceId, name, workspaceId !== "");

  if (workspaceId === "") {
    return (
      <RunFormState
        description="Select a workspace to run this Loop."
        testId="loop-run-form-no-workspace"
        title="No workspace selected"
      />
    );
  }
  if (loopQuery.isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="loop-run-form-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }
  if (loopQuery.error || !loopQuery.data) {
    return (
      <RunFormState
        description={loopQuery.error?.message ?? `Loop ${name} not found.`}
        icon={AlertCircle}
        testId="loop-run-form-error"
        title="Unable to load loop"
      />
    );
  }

  return (
    <LoopRunForm
      key={loopQuery.data.name}
      workspaceId={workspaceId}
      loop={loopQuery.data}
      onCancel={() => navigate({ to: "/loops/$name", params: { name } })}
      onRunStarted={runId => navigate({ to: "/loop-runs/$runId", params: { runId } })}
    />
  );
}

interface RunFormStateProps {
  title: string;
  description: string;
  testId: string;
  icon?: typeof Repeat2;
}

function RunFormState({ title, description, testId, icon = Repeat2 }: RunFormStateProps) {
  return (
    <div
      className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
      data-testid={testId}
    >
      <Empty className="max-w-md" description={description} icon={icon} title={title} />
    </div>
  );
}
