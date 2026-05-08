import type { ComponentProps } from "react";
import { AlertCircle, Loader2, Plus, type LucideIcon } from "lucide-react";

import { Button, Empty, PageHeader, PillGroup, SplitPane } from "@agh/ui";

import { AutomationDetailPanel } from "./automation-detail-panel";
import { AutomationEditorDialog } from "./automation-editor-dialog";
import { AutomationListPanel } from "./automation-list-panel";
import type { AutomationScopeFilter } from "../types";

interface AutomationOperationsPageProps {
  createButtonTestId: string;
  createLabel: string;
  icon: LucideIcon;
  page: {
    currentTotalCount: number;
    detailPanelProps: ComponentProps<typeof AutomationDetailPanel>;
    editorDialogProps: ComponentProps<typeof AutomationEditorDialog>;
    handleCreate: () => void;
    handleScopeChange: (nextScope: AutomationScopeFilter) => void;
    initialError: Error | null;
    isInitialLoading: boolean;
    listPanelProps: ComponentProps<typeof AutomationListPanel>;
    scopeFilter: AutomationScopeFilter;
  };
  title: string;
  titlePrefix: "jobs" | "triggers";
}

export function AutomationOperationsPage({
  createButtonTestId,
  createLabel,
  icon: Icon,
  page,
  title,
  titlePrefix,
}: AutomationOperationsPageProps) {
  if (page.isInitialLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid={`${titlePrefix}-loading`}
      >
        <Loader2 aria-hidden="true" className="size-5 animate-spin text-(--color-text-tertiary)" />
      </div>
    );
  }

  if (page.initialError) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid={`${titlePrefix}-error`}
      >
        <Empty
          className="max-w-md"
          description={page.initialError.message ?? `Failed to load ${title.toLowerCase()}`}
          icon={AlertCircle}
          title={`Unable to load ${title.toLowerCase()}`}
        />
      </div>
    );
  }

  const primaryAction = (
    <Button
      data-testid={createButtonTestId}
      onClick={page.handleCreate}
      size="sm"
      type="button"
      variant="outline"
    >
      <Plus className="size-3.5" />
      {createLabel}
    </Button>
  );

  return (
    <>
      <div
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
        data-testid={`${titlePrefix}-shell`}
      >
        <PageHeader
          count={page.currentTotalCount}
          controls={
            <PillGroup
              aria-label={`${title} scope`}
              data-testid={`${titlePrefix}-scope-tabs`}
              items={[
                { value: "all", label: "ALL", testId: `${titlePrefix}-scope-all` },
                { value: "global", label: "GLOBAL", testId: `${titlePrefix}-scope-global` },
                {
                  value: "workspace",
                  label: "WORKSPACE",
                  testId: `${titlePrefix}-scope-workspace`,
                },
              ]}
              onChange={page.handleScopeChange}
              value={page.scopeFilter}
            />
          }
          icon={() => <Icon className="size-3.5" data-testid={`${titlePrefix}-shell-icon`} />}
          meta={primaryAction}
          title={<span data-testid={`${titlePrefix}-shell-title`}>{title}</span>}
        />

        <SplitPane
          data-testid={`${titlePrefix}-split-pane`}
          detail={<AutomationDetailPanel {...page.detailPanelProps} />}
          list={<AutomationListPanel {...page.listPanelProps} />}
        />
      </div>

      <AutomationEditorDialog {...page.editorDialogProps} />
    </>
  );
}
