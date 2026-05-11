import { createFileRoute } from "@tanstack/react-router";
import { AlertCircle, BookOpen, Plus } from "lucide-react";

import { useKnowledgePage } from "@/hooks/routes/use-knowledge-page";
import {
  KnowledgeCreateDialog,
  KnowledgeDetailPanel,
  KnowledgeListPanel,
} from "@/systems/knowledge";
import type { TopbarRouteContext } from "@/types/topbar";
import { Button, Empty, Input, PillGroup, Spinner, SplitPane, useTopbarSlot } from "@agh/ui";

export const Route = createFileRoute("/_app/knowledge")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Knowledge", icon: BookOpen },
  }),
  component: KnowledgePage,
});

export function KnowledgePage() {
  const page = useKnowledgePage();

  const scopePills = (
    <PillGroup
      aria-label="Knowledge scope"
      data-testid="tab-pills"
      items={[
        { value: "global", label: "GLOBAL", testId: "tab-global" },
        { value: "workspace", label: "WORKSPACE", testId: "tab-workspace" },
        { value: "agent", label: "AGENT", testId: "tab-agent" },
      ]}
      onChange={value => page.setActiveScope(value as typeof page.activeScope)}
      value={page.activeScope}
    />
  );

  const agentControls =
    page.activeScope === "agent" ? (
      <div className="flex items-center gap-2" data-testid="agent-scope-controls">
        <Input
          aria-label="Agent name"
          className="h-7 w-44"
          data-testid="agent-name-input"
          onChange={event => page.setAgentName(event.target.value)}
          placeholder="agent name"
          value={page.agentName}
        />
        <PillGroup
          aria-label="Agent tier"
          data-testid="agent-tier-pills"
          items={[
            { value: "workspace", label: "WORKSPACE", testId: "tier-workspace" },
            { value: "global", label: "GLOBAL", testId: "tier-global" },
          ]}
          onChange={value => page.setAgentTier(value as typeof page.agentTier)}
          value={page.agentTier}
        />
      </div>
    ) : null;

  const createBtn = (
    <Button
      data-testid="create-memory-btn"
      disabled={!page.canCreateMemory}
      onClick={() => page.setCreateOpen(true)}
      size="sm"
      type="button"
      variant="outline"
    >
      <Plus className="size-3" />
      Create
    </Button>
  );

  useTopbarSlot({
    count: page.guardMessage || page.isLoading || page.error ? undefined : page.memoryCount,
    tabs: scopePills,
    search: agentControls,
    actions: createBtn,
  });

  if (page.guardMessage) {
    return (
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="knowledge-shell">
        <div
          className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
          data-testid="knowledge-guard"
        >
          <Empty
            className="max-w-md"
            description={page.guardMessage}
            icon={BookOpen}
            title="Select scope inputs"
          />
        </div>
      </div>
    );
  }

  if (page.isLoading) {
    return (
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="knowledge-shell">
        <div
          className="flex min-h-0 flex-1 items-center justify-center"
          data-testid="knowledge-loading"
        >
          <Spinner aria-hidden="true" className="size-5 text-subtle" />
        </div>
      </div>
    );
  }

  if (page.error) {
    return (
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="knowledge-shell">
        <div
          className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
          data-testid="knowledge-error"
        >
          <Empty
            className="max-w-md"
            description={page.error.message ?? "Failed to load knowledge"}
            icon={AlertCircle}
            title="Unable to load knowledge"
          />
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="knowledge-shell">
      <SplitPane
        data-testid="knowledge-split-pane"
        detail={
          <KnowledgeDetailPanel
            content={page.selectedContent}
            decisions={page.decisions}
            decisionsError={page.decisionsError}
            deleteError={page.deleteError}
            editError={page.editError}
            error={page.contentError}
            memory={page.selectedMemory}
            onDelete={page.handleDelete}
            onEdit={page.handleEdit}
            onRevertDecision={page.handleRevertDecision}
            revertError={page.revertError}
            revertingDecisionId={page.revertingDecisionId}
            scope={page.selectedScope}
            status={{
              isDecisionsLoading: page.isDecisionsLoading,
              isDeletePending: page.isDeletePending,
              isEditPending: page.isEditPending,
              isLoading: page.isContentLoading,
            }}
          />
        }
        list={
          <KnowledgeListPanel
            memories={page.memories}
            onSearchChange={page.setSearchQuery}
            onSelectMemory={page.setSelectedMemoryKey}
            searchInfo={page.searchInfo}
            searchMode={page.searchActive}
            searchQuery={page.searchQuery}
            selectedMemoryKey={page.effectiveSelectedMemoryKey}
          />
        }
      />
      <KnowledgeCreateDialog
        defaultType={page.createDefaultType}
        error={page.createError}
        isPending={page.isCreatePending}
        onConfirm={page.handleCreate}
        onOpenChange={page.setCreateOpen}
        open={page.createOpen}
        scope={page.activeScope}
      />
    </div>
  );
}
