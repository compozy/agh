import { useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { AlertCircle, Book, Loader2 } from "lucide-react";

import { PillButton } from "@/components/design-system";
import {
  useMemories,
  useMemory,
  useDeleteMemory,
  KnowledgeListPanel,
  KnowledgeDetailPanel,
} from "@/systems/knowledge";
import type { MemoryScope } from "@/systems/knowledge/types";
import { useActiveWorkspace } from "@/systems/workspace";
import { WorkspacePageShell } from "@/systems/workspace/components/workspace-page-shell";

export const Route = createFileRoute("/_app/knowledge")({
  component: KnowledgePage,
});

// ---------------------------------------------------------------------------
// Tab type
// ---------------------------------------------------------------------------

type Tab = "all" | "global" | "workspace";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function deriveScope(filename: string): Exclude<MemoryScope, undefined> {
  if (filename.startsWith("workspace/") || filename.startsWith("ws/")) {
    return "workspace";
  }
  return "global";
}

function formatDreamStatus(lastConsolidation?: string): string {
  if (!lastConsolidation) return "Dream: status unavailable";
  try {
    const date = new Date(lastConsolidation);
    const diffMs = Date.now() - date.getTime();
    const diffH = Math.floor(diffMs / (1000 * 60 * 60));
    if (diffH < 1) return "Dream: <1h ago";
    if (diffH < 24) return `Dream: ${diffH}h ago`;
    const diffD = Math.floor(diffH / 24);
    return `Dream: ${diffD}d ago`;
  } catch {
    return "Dream: unknown";
  }
}

// ---------------------------------------------------------------------------
// Knowledge Page
// ---------------------------------------------------------------------------

function KnowledgePage() {
  const [activeTab, setActiveTab] = useState<Tab>("all");
  const [selectedFilename, setSelectedFilename] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  const { activeWorkspaceId } = useActiveWorkspace();

  // Determine scope filter for API
  const scopeFilter = activeTab === "all" ? undefined : activeTab;

  // Data hooks
  const {
    data: memories,
    isLoading,
    error,
  } = useMemories(scopeFilter, activeWorkspaceId || undefined);

  const deleteMutation = useDeleteMemory();

  const memoryCount = memories?.length ?? 0;

  // Auto-select first memory if none selected
  const effectiveSelectedFilename = useMemo(() => {
    if (selectedFilename && memories?.some(m => m.filename === selectedFilename)) {
      return selectedFilename;
    }
    return memories?.[0]?.filename ?? null;
  }, [selectedFilename, memories]);

  // Find the selected memory header
  const selectedMemory = useMemo(
    () => memories?.find(m => m.filename === effectiveSelectedFilename),
    [memories, effectiveSelectedFilename]
  );

  // Load content for selected memory
  const selectedScope = effectiveSelectedFilename
    ? deriveScope(effectiveSelectedFilename)
    : undefined;
  const {
    data: selectedContent,
    isLoading: isContentLoading,
    error: contentError,
  } = useMemory(selectedScope, effectiveSelectedFilename ?? "", activeWorkspaceId || undefined);

  const handleDelete = (filename: string) => {
    if (!selectedScope) {
      return;
    }
    deleteMutation.mutate({
      scope: selectedScope,
      filename,
      workspace: activeWorkspaceId || undefined,
    });
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="knowledge-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="knowledge-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {error.message ?? "Failed to load knowledge"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <WorkspacePageShell
      title="Knowledge"
      icon={<Book className="size-4" />}
      count={memoryCount}
      controls={
        <div className="flex items-center gap-1.5" data-testid="tab-pills">
          {(["all", "global", "workspace"] as const).map(tab => (
            <PillButton
              key={tab}
              active={activeTab === tab}
              data-testid={`tab-${tab}`}
              onClick={() => setActiveTab(tab)}
            >
              {tab.toUpperCase()}
            </PillButton>
          ))}
        </div>
      }
      meta={
        <div className="flex items-center gap-1.5" data-testid="dream-status">
          <span className="size-2 rounded-full bg-[color:var(--color-text-tertiary)]" />
          <span className="text-xs text-[color:var(--color-text-tertiary)]">
            {formatDreamStatus()}
          </span>
        </div>
      }
    >
      <KnowledgeListPanel
        memories={memories ?? []}
        selectedFilename={effectiveSelectedFilename}
        onSelectMemory={setSelectedFilename}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
      />
      <KnowledgeDetailPanel
        memory={selectedMemory}
        content={selectedContent}
        scope={selectedScope}
        isLoading={isContentLoading && effectiveSelectedFilename !== null}
        error={contentError}
        onDelete={handleDelete}
        isDeletePending={deleteMutation.isPending}
      />
    </WorkspacePageShell>
  );
}
