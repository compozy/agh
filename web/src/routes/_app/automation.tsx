import { AlertCircle, Loader2, Plus, Zap } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Button, Pills } from "@agh/ui";
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
      <div className="flex flex-1 items-center justify-center" data-testid="automation-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.initialError) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="automation-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.initialError.message ?? "Failed to load automation"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <header className="flex flex-wrap items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3">
        <div className="flex items-center gap-2">
          <Zap className="size-4 text-[color:var(--color-text-primary)]" />
          <h1 className="text-xl font-semibold tracking-[-0.02em] text-[color:var(--color-text-primary)]">
            Automation
          </h1>
          <span className="inline-flex h-5 items-center rounded-md bg-[color:var(--color-surface-panel)] px-1.5 font-mono text-[0.64rem] text-[color:var(--color-text-secondary)]">
            {page.currentTotalCount}
          </span>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <Pills
            data-testid="automation-kind-tabs"
            value={page.activeTab}
            onChange={page.handleTabChange}
            items={[
              { value: "jobs", label: "JOBS", testId: "automation-kind-jobs" },
              { value: "triggers", label: "TRIGGERS", testId: "automation-kind-triggers" },
            ]}
          />

          <Pills
            data-testid="automation-scope-tabs"
            value={page.scopeFilter}
            onChange={page.handleScopeChange}
            items={[
              { value: "all", label: "ALL", testId: "automation-scope-all" },
              { value: "global", label: "GLOBAL", testId: "automation-scope-global" },
              { value: "workspace", label: "WORKSPACE", testId: "automation-scope-workspace" },
            ]}
          />
        </div>

        <div className="ml-auto flex items-center gap-2">
          <Button
            className="border-[color:var(--color-divider)] bg-transparent text-[color:var(--color-text-primary)] hover:bg-[color:var(--color-hover)]"
            data-testid="create-automation-btn"
            onClick={page.handleCreate}
            size="lg"
            type="button"
            variant="outline"
          >
            <Plus className="size-4" />
            {page.activeTab === "jobs" ? "Job" : "Trigger"}
          </Button>
        </div>
      </header>

      <div className="flex min-h-0 flex-1 overflow-hidden">
        <AutomationListPanel {...page.listPanelProps} />
        <AutomationDetailPanel {...page.detailPanelProps} />
      </div>

      <AutomationEditorDialog {...page.editorDialogProps} />
    </div>
  );
}
