import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { AutomationDetailPanel, AutomationEditorDialog } from "@/systems/automation";
import { useAutomationTriggerDetailPage } from "@/hooks/routes/use-automation-page";
import { preloadAutomationTriggerDetailRoute } from "./-automation-preload";

export const Route = createFileRoute("/_app/triggers/$triggerId")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    // Parent `/triggers` crumb already supplies the Triggers link — do not re-add parentCrumb.
    topbar: { crumb: { label: params.triggerId } },
  }),
  loader: ({ context, params }) =>
    preloadAutomationTriggerDetailRoute(context.queryClient, params.triggerId),
  component: TriggerDetailRoute,
});

function TriggerDetailRoute() {
  const { triggerId } = Route.useParams();
  const page = useAutomationTriggerDetailPage(triggerId);

  return (
    <>
      <AutomationDetailPanel
        error={page.error}
        item={page.trigger}
        kind="triggers"
        onDelete={page.handleDelete}
        onEdit={page.handleEdit}
        onToggleEnabled={page.handleToggleEnabled}
        runs={page.runs}
        runsError={page.runsError}
        runsLoading={page.runsLoading}
        state={{
          isDeleting: page.isDeleting,
          isLoading: page.isLoading,
          isTogglePending: page.isTogglePending,
          isTriggerPending: false,
        }}
      />

      <AutomationEditorDialog {...page.editorDialogProps} />
    </>
  );
}
