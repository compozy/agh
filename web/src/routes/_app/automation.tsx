import { AlertCircle, Loader2, Plus, Zap } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Button, Empty, PageHeader, Pills, SplitPane } from "@agh/ui";
import {
  AutomationDetailPanel,
  AutomationEditorDialog,
  AutomationListPanel,
} from "@/systems/automation";
import { useAutomationPage } from "@/hooks/routes/use-automation-page";

export const Route = createFileRoute("/_app/automation")({
  component: AutomationPage,
});

function AutomationPage() {
  const page = useAutomationPage();

  if (page.isInitialLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="automation-loading"
      >
        <Loader2
          aria-hidden="true"
          className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
        />
      </div>
    );
  }

  if (page.initialError) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="automation-error"
      >
        <Empty
          className="max-w-md"
          description={page.initialError.message ?? "Failed to load automation"}
          icon={AlertCircle}
          title="Unable to load automation"
        />
      </div>
    );
  }

  const primaryAction = (
    <Button
      data-testid="create-automation-btn"
      onClick={page.handleCreate}
      size="sm"
      type="button"
      variant="outline"
    >
      <Plus className="size-3.5" />
      {page.activeTab === "jobs" ? "Job" : "Trigger"}
    </Button>
  );

  return (
    <>
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="automation-shell">
        <PageHeader
          count={page.currentTotalCount}
          controls={
            <div className="flex flex-wrap items-center gap-2">
              <Pills
                aria-label="Automation kind"
                data-testid="automation-kind-tabs"
                items={[
                  { value: "jobs", label: "JOBS", testId: "automation-kind-jobs" },
                  { value: "triggers", label: "TRIGGERS", testId: "automation-kind-triggers" },
                ]}
                onChange={page.handleTabChange}
                value={page.activeTab}
              />
              <Pills
                aria-label="Automation scope"
                data-testid="automation-scope-tabs"
                items={[
                  { value: "all", label: "ALL", testId: "automation-scope-all" },
                  { value: "global", label: "GLOBAL", testId: "automation-scope-global" },
                  { value: "workspace", label: "WORKSPACE", testId: "automation-scope-workspace" },
                ]}
                onChange={page.handleScopeChange}
                value={page.scopeFilter}
              />
            </div>
          }
          icon={() => <Zap className="size-3.5" data-testid="automation-shell-icon" />}
          meta={primaryAction}
          title={<span data-testid="automation-shell-title">Automation</span>}
        />

        <SplitPane
          data-testid="automation-split-pane"
          detail={<AutomationDetailPanel {...page.detailPanelProps} />}
          list={<AutomationListPanel {...page.listPanelProps} />}
        />
      </div>

      <AutomationEditorDialog {...page.editorDialogProps} />
    </>
  );
}
