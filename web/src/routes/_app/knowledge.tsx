import { AlertCircle, BookOpen, Loader2 } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Empty, PageHeader, Pills, SplitPane, StatusDot } from "@agh/ui";
import { useKnowledgePage } from "@/hooks/routes/use-knowledge-page";
import { KnowledgeDetailPanel, KnowledgeListPanel } from "@/systems/knowledge";

export const Route = createFileRoute("/_app/knowledge")({
  component: KnowledgePage,
});

function KnowledgePage() {
  const page = useKnowledgePage();

  if (page.isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="knowledge-loading"
      >
        <Loader2
          aria-hidden="true"
          className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
        />
      </div>
    );
  }

  if (page.error) {
    return (
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
    );
  }

  const controls = (
    <Pills
      aria-label="Knowledge scope"
      data-testid="tab-pills"
      items={[
        { value: "all", label: "ALL", testId: "tab-all" },
        { value: "global", label: "GLOBAL", testId: "tab-global" },
        { value: "workspace", label: "WORKSPACE", testId: "tab-workspace" },
      ]}
      onChange={page.setActiveTab}
      value={page.activeTab}
    />
  );

  const dreamMeta = (
    <div className="flex items-center gap-1.5" data-testid="dream-status">
      <StatusDot tone="neutral" />
      <span className="font-mono text-[11px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
        {page.dreamStatusLabel}
      </span>
    </div>
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="knowledge-shell">
      <PageHeader
        count={page.memoryCount}
        controls={controls}
        icon={() => <BookOpen className="size-3.5" data-testid="knowledge-shell-icon" />}
        meta={dreamMeta}
        title={<span data-testid="knowledge-shell-title">Knowledge</span>}
      />
      <SplitPane
        data-testid="knowledge-split-pane"
        detail={
          <KnowledgeDetailPanel
            content={page.selectedContent}
            error={page.contentError}
            isDeletePending={page.isDeletePending}
            isLoading={page.isContentLoading}
            memory={page.selectedMemory}
            onDelete={page.handleDelete}
            scope={page.selectedScope}
          />
        }
        list={
          <KnowledgeListPanel
            memories={page.memories}
            onSearchChange={page.setSearchQuery}
            onSelectMemory={page.setSelectedFilename}
            searchQuery={page.searchQuery}
            selectedFilename={page.effectiveSelectedFilename}
          />
        }
      />
    </div>
  );
}
