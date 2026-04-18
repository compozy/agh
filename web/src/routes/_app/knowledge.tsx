import { AlertCircle, Book, Loader2 } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Pills } from "@agh/ui";
import { useKnowledgePage } from "@/hooks/routes/use-knowledge-page";
import { KnowledgeDetailPanel, KnowledgeListPanel } from "@/systems/knowledge";
import { WorkspacePageShell } from "@/systems/workspace/components/workspace-page-shell";

export const Route = createFileRoute("/_app/knowledge")({
  component: KnowledgePage,
});

function KnowledgePage() {
  const page = useKnowledgePage();

  if (page.isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="knowledge-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.error) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="knowledge-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.error.message ?? "Failed to load knowledge"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <WorkspacePageShell
      title="Knowledge"
      icon={<Book className="size-4" />}
      count={page.memoryCount}
      controls={
        <Pills
          data-testid="tab-pills"
          value={page.activeTab}
          onChange={page.setActiveTab}
          items={[
            { value: "all", label: "ALL", testId: "tab-all" },
            { value: "global", label: "GLOBAL", testId: "tab-global" },
            { value: "workspace", label: "WORKSPACE", testId: "tab-workspace" },
          ]}
        />
      }
      meta={
        <div className="flex items-center gap-1.5" data-testid="dream-status">
          <span className="size-2 rounded-full bg-[color:var(--color-text-tertiary)]" />
          <span className="text-xs text-[color:var(--color-text-tertiary)]">
            {page.dreamStatusLabel}
          </span>
        </div>
      }
    >
      <KnowledgeListPanel
        memories={page.memories}
        selectedFilename={page.effectiveSelectedFilename}
        onSelectMemory={page.setSelectedFilename}
        searchQuery={page.searchQuery}
        onSearchChange={page.setSearchQuery}
      />
      <KnowledgeDetailPanel
        memory={page.selectedMemory}
        content={page.selectedContent}
        scope={page.selectedScope}
        isLoading={page.isContentLoading}
        error={page.contentError}
        onDelete={page.handleDelete}
        isDeletePending={page.isDeletePending}
      />
    </WorkspacePageShell>
  );
}
