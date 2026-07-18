import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { AutomationDetailPanel, AutomationEditorDialog } from "@/systems/automation";
import { useAutomationJobDetailPage } from "@/hooks/routes/use-automation-page";
import { preloadAutomationJobDetailRoute } from "./-automation-preload";

export const Route = createFileRoute("/_app/jobs/$jobId")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    // Parent `/jobs` crumb already supplies the Jobs link — do not re-add parentCrumb.
    topbar: { crumb: { label: params.jobId } },
  }),
  loader: ({ context, params }) =>
    preloadAutomationJobDetailRoute(context.queryClient, params.jobId),
  component: JobDetailRoute,
});

function JobDetailRoute() {
  const { jobId } = Route.useParams();
  const page = useAutomationJobDetailPage(jobId);

  return (
    <>
      <AutomationDetailPanel
        error={page.error}
        item={page.job}
        kind="jobs"
        onDelete={page.handleDelete}
        onEdit={page.handleEdit}
        onToggleEnabled={page.handleToggleEnabled}
        onTriggerNow={page.handleTriggerNow}
        runs={page.runs}
        runsError={page.runsError}
        runsLoading={page.runsLoading}
        state={{
          isDeleting: page.isDeleting,
          isLoading: page.isLoading,
          isTogglePending: page.isTogglePending,
          isTriggerPending: page.isTriggerPending,
        }}
      />

      <AutomationEditorDialog {...page.editorDialogProps} />
    </>
  );
}
