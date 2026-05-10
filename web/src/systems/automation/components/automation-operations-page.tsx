import type { ComponentProps } from "react";
import { AlertCircle, Plus } from "lucide-react";

import { Button, Empty, PillGroup, Spinner, SplitPane, useTopbarSlot } from "@agh/ui";

import { AutomationDetailPanel } from "./automation-detail-panel";
import { AutomationEditorDialog } from "./automation-editor-dialog";
import { AutomationListPanel } from "./automation-list-panel";
import type { AutomationScopeFilter } from "../types";

interface AutomationOperationsPageProps {
  createButtonTestId: string;
  createLabel: string;
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
  page,
  title,
  titlePrefix,
}: AutomationOperationsPageProps) {
  useTopbarSlot({
    count: page.currentTotalCount,
    tabs: (
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
    ),
    actions: (
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
    ),
  });

  if (page.isInitialLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid={`${titlePrefix}-loading`}
      >
        <Spinner className="size-5 text-(--subtle)" />
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

  return (
    <>
      <div
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
        data-testid={`${titlePrefix}-shell`}
      >
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
