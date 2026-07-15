import { AlertCircle, Repeat2, SlidersHorizontal } from "lucide-react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { Empty, Spinner } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { LoopConfigureSheet, useLoop, useLoopConfig } from "@/systems/loops";
import { useActiveWorkspace } from "@/systems/workspace";
import { preloadLoopConfigureRoute } from "./-loops-preload";

export const Route = createFileRoute("/_app/loops/$name/configure")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `Configure ${params.name}`, icon: SlidersHorizontal },
  }),
  loader: ({ context, params }) => preloadLoopConfigureRoute(context.queryClient, params.name),
  component: LoopConfigureRoute,
});

/**
 * The no-fork Configure surface (design §4.7): a right-side sheet over a dimmed backdrop that
 * writes the per-loop `loop_config` store. Opened from the detail Configure action (task 19);
 * closing or saving returns to the loop detail. Structural edits route to the fork editor.
 */
function LoopConfigureRoute() {
  const { name } = Route.useParams();
  const navigate = useNavigate();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const loopQuery = useLoop(workspaceId, name, workspaceId !== "");
  const configQuery = useLoopConfig(workspaceId, name, workspaceId !== "");

  const close = () => void navigate({ to: "/loops/$name", params: { name } });

  if (workspaceId === "") {
    return (
      <ConfigureState
        description="Select a workspace to configure this Loop."
        testId="loop-configure-no-workspace"
        title="No workspace selected"
      />
    );
  }
  if (loopQuery.isLoading || configQuery.isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="loop-configure-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }
  // Gate on the config load failing too: `getLoopConfig` returns null only on 404 (an
  // expected "unconfigured" loop); a hard load error (500/network) must NOT open the sheet
  // on defaults, or a Save would silently overwrite the real stored config (R-003). Gate on
  // `isLoadingError` (first-fetch failure with no cached data), so a transient background
  // refetch failure keeps the sheet mounted over stale-but-valid config instead of tearing
  // it down and discarding the operator's in-progress edits (R-001, round 2).
  if (loopQuery.isLoadingError || !loopQuery.data || configQuery.isLoadingError) {
    const message =
      loopQuery.error?.message ?? configQuery.error?.message ?? `Loop ${name} not found.`;
    return (
      <ConfigureState
        description={message}
        icon={AlertCircle}
        testId="loop-configure-error"
        title="Unable to load loop"
      />
    );
  }

  if (!configQuery.effectiveConfig) {
    return (
      <ConfigureState
        description="The daemon did not return effective loop configuration."
        icon={AlertCircle}
        testId="loop-configure-effective-error"
        title="Unable to load loop configuration"
      />
    );
  }

  return (
    <LoopConfigureSheet
      key={name}
      open
      workspaceId={workspaceId}
      loop={loopQuery.data}
      config={configQuery.data ?? null}
      effectiveConfig={configQuery.effectiveConfig}
      onOpenChange={next => {
        if (!next) close();
      }}
      onOpenEditor={() => void navigate({ to: "/loops/$name/editor", params: { name } })}
    />
  );
}

interface ConfigureStateProps {
  title: string;
  description: string;
  testId: string;
  icon?: typeof Repeat2;
}

function ConfigureState({ title, description, testId, icon = Repeat2 }: ConfigureStateProps) {
  return (
    <div
      className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
      data-testid={testId}
    >
      <Empty className="max-w-md" description={description} icon={icon} title={title} />
    </div>
  );
}
